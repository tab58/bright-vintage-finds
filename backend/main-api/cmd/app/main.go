package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"main-api/api"
	"main-api/cmd/app/config"
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

	// ponytail: storage skipped entirely when S3_BASE_ENDPOINT is unset, so
	// environments without a bucket still boot; make it required once an
	// upload feature depends on it
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
