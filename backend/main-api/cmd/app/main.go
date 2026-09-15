package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"main-api/api"
	"main-api/cmd/app/config"
	db_platform "main-api/db"
	"main-api/internal/cfaccess"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	aws_s3 "github.com/tab58/bright-vintage-finds/environment/shared/golang/clients/aws_s3"
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

	deps := &api.AppDeps{}

	// Database (required in production; optional in development so an empty
	// env still boots for healthcheck smoke tests).
	if cfg.MainDBURL != "" {
		client, err := db_platform.NewClient(db_platform.ClientConfig{ConnectionString: cfg.MainDBURL})
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		deps.DB = client

		// Seed the builtin selling places; fail-fast so a broken seed surfaces
		// at deploy time, not when the intake UI first loads.
		if err := api.SeedBuiltinSellingPlaces(client); err != nil {
			return fmt.Errorf("failed to seed selling places: %w", err)
		}
	}

	// Object storage (optional; the image upload route needs it).
	if cfg.S3BaseEndpoint != "" {
		if cfg.S3UploadBucket == "" {
			return fmt.Errorf("S3_UPLOAD_BUCKET is required when S3_BASE_ENDPOINT is set")
		}
		awsCfg, err := awsconfig.LoadDefaultConfig(context.Background())
		if err != nil {
			return fmt.Errorf("failed to load AWS config: %w", err)
		}
		store := aws_s3.NewClient(awsCfg, aws_s3.WithBaseEndpoint(cfg.S3BaseEndpoint))
		pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := store.Ping(pingCtx, cfg.S3UploadBucket); err != nil {
			return fmt.Errorf("failed to reach object storage: %w", err)
		}
		deps.Store = store
		deps.S3UploadBucket = cfg.S3UploadBucket
		deps.S3InternalEndpoint = cfg.S3BaseEndpoint
		deps.S3PublicEndpoint = cfg.S3PublicEndpoint
	}

	srv := api.NewServer(server.ServerConfig{
		ServiceName:        "main-api",
		ServiceVersion:     "1.0.0",
		ServiceDescription: "Main API for the application",
		Environment:        cfg.Env,
	}, router.MapAuthInfoBuilder, deps, server.WithMiddleware(adminGuard))

	errCh, err := srv.Start(":" + cfg.ServerPort)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	fmt.Println("listening on :" + cfg.ServerPort)

	// ponytail: no signal-based graceful shutdown yet; add signal.NotifyContext
	// + srv.Shutdown when deploys need drain
	return <-errCh
}
