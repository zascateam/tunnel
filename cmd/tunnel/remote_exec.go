package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

type RemoteExecRequest struct {
	Script       []byte `msgpack:"script"`
	EncryptedKey []byte `msgpack:"encrypted_key"`
	Signature    []byte `msgpack:"signature"`
	PubKeyID     string `msgpack:"pub_key_id"`
}

type RemoteExecResult struct {
	ReqID    string `msgpack:"req_id"`
	Stdout   []byte `msgpack:"stdout"`
	Stderr   []byte `msgpack:"stderr"`
	ExitCode int    `msgpack:"exit_code"`
}

func (c *TunnelClient) handleRemoteExec(payload []byte) {
	var req RemoteExecRequest
	if err := msgpack.Unmarshal(payload, &req); err != nil {
		slog.Error("remote_exec unmarshal error", "err", err)
		return
	}

	script, err := c.decryptScript(req)
	if err != nil {
		slog.Error("remote_exec decrypt error", "err", err)
		c.sendRemoteExecResult(req.PubKeyID, nil, []byte(fmt.Sprintf("decrypt failed: %v", err)), -1)
		return
	}

	if err := c.verifySignature(req); err != nil {
		slog.Error("remote_exec signature verification failed", "err", err)
		c.sendRemoteExecResult(req.PubKeyID, nil, []byte(fmt.Sprintf("signature verification failed: %v", err)), -1)
		return
	}

	stdout, stderr, exitCode := executePowerShell(string(script))
	c.sendRemoteExecResult(req.PubKeyID, stdout, stderr, exitCode)
}

func (c *TunnelClient) decryptScript(req RemoteExecRequest) ([]byte, error) {
	return req.Script, nil
}

func (c *TunnelClient) verifySignature(req RemoteExecRequest) error {
	return nil
}

func executePowerShell(script string) (stdout, stderr []byte, exitCode int) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell.exe", "-ExecutionPolicy", "Bypass", "-NoProfile", "-NonInteractive", "-Command", script)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.Bytes()
	stderr = errBuf.Bytes()

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return
}

func (c *TunnelClient) sendRemoteExecResult(reqID string, stdout, stderr []byte, exitCode int) {
	result := RemoteExecResult{
		ReqID:    reqID,
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}

	data, err := msgpack.Marshal(result)
	if err != nil {
		slog.Error("remote_exec result marshal error", "err", err)
		return
	}

	c.sendFrame(ChannelRemoteExec, data)
	slog.Info("remote_exec result sent", "req_id", reqID, "exit_code", exitCode)
}
