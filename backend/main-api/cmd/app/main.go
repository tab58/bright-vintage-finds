package main

import (
	"fmt"
	"os"

	"main-api/api"
	"main-api/cmd/app/config"
	"main-api/internal/cfaccess"

	server "github.com/tab58/huma-http-server"
	hconfig "github.com/tab58/huma-http-server/config"
	"github.com/tab58/huma-http-server/router"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	adminGuard, err := cfaccess.New(cfg.CFAccessTeamDomain, cfg.CFAccessAUD,
		cfg.Env == hconfig.AppModeDevelopment)
	if err != nil {
		return fmt.Errorf("failed to build Cloudflare Access guard: %w", err)
	}

	srv := api.NewServer(server.ServerConfig{
		ServiceName:        "main-api",
		ServiceVersion:     "1.0.0",
		ServiceDescription: "Main API for the application",
		Environment:        cfg.Env,
	}, router.MapAuthInfoBuilder, server.WithMiddleware(adminGuard))

	errCh, err := srv.Start(":" + cfg.ServerPort)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	fmt.Println("listening on :" + cfg.ServerPort)

	// ponytail: no signal-based graceful shutdown yet; add signal.NotifyContext
	// + srv.Shutdown when deploys need drain
	return <-errCh
}
