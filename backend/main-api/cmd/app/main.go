package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"main-api/api"
	"main-api/cmd/app/config"
	db_platform "main-api/db"
	"main-api/internal/app"
	entadapter "main-api/internal/app/adapters/ent"
	s3adapter "main-api/internal/app/adapters/s3"
	"main-api/internal/app/ports"
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

	// Object storage (optional; the photo routes need it). It is built before
	// the application so the wiring sees it.
	var store ports.ImageStore
	if cfg.S3BaseEndpoint != "" {
		store, err = imageStore(cfg)
		if err != nil {
			return err
		}
	}

	// Database (required in production; optional in development so an empty
	// env still boots for healthcheck smoke tests). Without one there is no
	// application at all, and only the platform routes are served.
	var application *app.Application
	if cfg.MainDBURL != "" {
		application, err = inventoryApp(cfg.MainDBURL, store)
		if err != nil {
			return err
		}

		// Seed the builtin selling places; fail-fast so a broken seed surfaces
		// at deploy time, not when the intake UI first loads.
		if err := application.SeedBuiltinPlaces(context.Background()); err != nil {
			return fmt.Errorf("failed to seed selling places: %w", err)
		}
	}

	srv := api.NewServer(server.ServerConfig{
		ServiceName:        "main-api",
		ServiceVersion:     "1.0.0",
		ServiceDescription: "Main API for the application",
		Environment:        cfg.Env,
	}, router.MapAuthInfoBuilder, application, server.WithMiddleware(adminGuard))

	errCh, err := srv.Start(":" + cfg.ServerPort)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	fmt.Println("listening on :" + cfg.ServerPort)

	// ponytail: no signal-based graceful shutdown yet; add signal.NotifyContext
	// + srv.Shutdown when deploys need drain
	return <-errCh
}

// imageStore builds the object storage adapter and checks the bucket is
// reachable, so a misconfigured deployment fails at boot rather than on the
// first upload.
func imageStore(cfg *config.Config) (ports.ImageStore, error) {
	if cfg.S3UploadBucket == "" {
		return nil, fmt.Errorf("S3_UPLOAD_BUCKET is required when S3_BASE_ENDPOINT is set")
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := aws_s3.NewClient(awsCfg, aws_s3.WithBaseEndpoint(cfg.S3BaseEndpoint))

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx, cfg.S3UploadBucket); err != nil {
		return nil, fmt.Errorf("failed to reach object storage: %w", err)
	}

	return s3adapter.New(client, s3adapter.Config{
		Bucket:           cfg.S3UploadBucket,
		InternalEndpoint: cfg.S3BaseEndpoint,
		PublicEndpoint:   cfg.S3PublicEndpoint,
	}), nil
}

// inventoryApp opens the inventory database and wires the application over the
// Ent repositories. This is the composition root: the only place that knows
// both which adapters exist and which ports they fill.
func inventoryApp(dbURL string, store ports.ImageStore) (*app.Application, error) {
	client, err := db_platform.NewClient(db_platform.ClientConfig{ConnectionString: dbURL})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	gen := client.GetDBFromContext(context.Background())

	return app.New(app.Config{
		Items:  entadapter.NewItemRepo(gen),
		Images: entadapter.NewItemImageRepo(gen),
		Labels: entadapter.NewLabelRepo(gen),
		Places: entadapter.NewSellingPlaceRepo(gen),
		Owner:  entadapter.NewOwnerRepo(gen),
		Store:  store,
	}), nil
}
