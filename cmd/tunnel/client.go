package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	ChannelRDP        byte = 0x01
	ChannelWinRM      byte = 0x02
	ChannelRemoteExec byte = 0x03
	ChannelControl    byte = 0xFF
)

type TunnelClient struct {
	cfg       *TunnelConfig
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey

	mu       sync.Mutex
	conn     *websocket.Conn
	closed   bool
	closeCh  chan struct{}

	rdpConn   net.Conn
	winrmConn net.Conn
}

func NewTunnelClient(cfg *TunnelConfig) (*TunnelClient, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	return &TunnelClient{
		cfg:        cfg,
		privateKey: priv,
		publicKey:  pub,
		closeCh:    make(chan struct{}),
	}, nil
}

func (c *TunnelClient) PublicKey() string {
	return base64.StdEncoding.EncodeToString(c.publicKey)
}

func (c *TunnelClient) Connect(ctx context.Context) error {
	u, err := url.Parse(c.cfg.Server)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	q := u.Query()
	q.Set("token", c.cfg.Token)
	q.Set("ver", "1.0.0")
	q.Set("pubkey", c.PublicKey())
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
	}

	headers := http.Header{}
	conn, _, err := dialer.DialContext(ctx, u.String(), headers)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.closed = false
	c.mu.Unlock()

	slog.Info("connected to gateway", "server", c.cfg.Server)
	return nil
}

func (c *TunnelClient) Run(ctx context.Context) error {
	backoff := 1 * time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := c.Connect(ctx); err != nil {
			slog.Error("connection failed", "err", err, "retry_in", backoff)
			select {
			case <-time.After(backoff):
				backoff = min(backoff*2, maxBackoff)
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		backoff = 1 * time.Second

		heartbeat := time.NewTicker(30 * time.Second)
		defer heartbeat.Stop()

		go c.sendHeartbeat(heartbeat.C)

		err := c.readLoop()
		c.Close()

		if err != nil {
			slog.Error("tunnel disconnected", "err", err)
		}

		select {
		case <-time.After(backoff):
			backoff = min(backoff*2, maxBackoff)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *TunnelClient) readLoop() error {
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return fmt.Errorf("connection nil")
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		if len(message) < 3 {
			continue
		}

		length := uint16(message[0])<<8 | uint16(message[1])
		channel := message[2]
		payload := message[3:]

		_ = length

		switch channel {
		case ChannelRDP:
			go c.forwardRDP(payload)
		case ChannelWinRM:
			go c.forwardWinRM(payload)
		case ChannelRemoteExec:
			go c.handleRemoteExec(payload)
		case ChannelControl:
			slog.Debug("control frame received", "len", len(payload))
		default:
			slog.Warn("unknown channel", "channel", channel)
		}
	}
}

func (c *TunnelClient) sendHeartbeat(ticker <-chan time.Time) {
	for range ticker {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		frame := []byte{0x00, 0x00, ChannelControl}
		if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
			slog.Error("heartbeat send failed", "err", err)
			return
		}
	}
}

func (c *TunnelClient) forwardRDP(payload []byte) {
	conn, err := net.DialTimeout("tcp", c.cfg.RDP, 5*time.Second)
	if err != nil {
		slog.Error("RDP connect failed", "err", err)
		return
	}
	defer conn.Close()

	if _, err := conn.Write(payload); err != nil {
		slog.Error("RDP write failed", "err", err)
		return
	}

	buf := make([]byte, 32*1024)
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	c.sendFrame(ChannelRDP, buf[:n])
}

func (c *TunnelClient) forwardWinRM(payload []byte) {
	conn, err := net.DialTimeout("tcp", c.cfg.WinRM, 5*time.Second)
	if err != nil {
		slog.Error("WinRM connect failed", "err", err)
		return
	}
	defer conn.Close()

	if _, err := conn.Write(payload); err != nil {
		return
	}

	buf := make([]byte, 32*1024)
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	c.sendFrame(ChannelWinRM, buf[:n])
}

func (c *TunnelClient) sendFrame(channel byte, payload []byte) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}

	length := uint16(len(payload))
	frame := make([]byte, 3+len(payload))
	frame[0] = byte(length >> 8)
	frame[1] = byte(length)
	frame[2] = channel
	copy(frame[3:], payload)

	if err := conn.WriteMessage(websocket.BinaryMessage, frame); err != nil {
		slog.Error("frame send failed", "err", err)
	}
}

func (c *TunnelClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && !c.closed {
		c.conn.Close()
		c.closed = true
	}
}

func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
