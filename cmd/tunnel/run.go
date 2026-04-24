package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func runService(configPath string) error {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	slog.Info("ZASCA Tunnel starting",
		"server", cfg.Server,
		"rdp", cfg.RDP,
		"winrm", cfg.WinRM,
	)

	client, err := NewTunnelClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create tunnel client: %w", err)
	}

	slog.Info("Ed25519 public key generated", "pubkey", client.PublicKey())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("shutdown signal received")
		cancel()
	}()

	if err := client.Run(ctx); err != nil {
		if ctx.Err() != nil {
			slog.Info("ZASCA Tunnel stopped")
			return nil
		}
		return fmt.Errorf("tunnel run error: %w", err)
	}

	return nil
}
