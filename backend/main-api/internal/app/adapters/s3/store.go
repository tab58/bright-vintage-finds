// Package s3 implements the ImageStore port over the shared AWS S3 client.
package s3

import (
	"context"
	"io"
	"strings"
	"time"

	"main-api/internal/app/domain"
	"main-api/internal/app/ports"

	aws_s3 "github.com/tab58/bright-vintage-finds/environment/shared/golang/clients/aws_s3"
)

var _ ports.ImageStore = (*Store)(nil)

// Store is object storage for item photos.
type Store struct {
	client aws_s3.Client
	bucket string
	// internalEndpoint is the base URL presigned URLs are minted against, and
	// publicEndpoint is the one that resolves outside the storage's network.
	// Both are set only in local development.
	internalEndpoint string
	publicEndpoint   string
}

// Config carries the deployment's storage settings.
type Config struct {
	// Bucket is where new uploads go.
	Bucket string
	// InternalEndpoint is S3_BASE_ENDPOINT, the host presigned URLs carry.
	InternalEndpoint string
	// PublicEndpoint, when set, replaces that host so the URL resolves from
	// outside the storage's internal network. Local development only.
	PublicEndpoint string
}

// New returns a store over the given S3 client.
func New(client aws_s3.Client, cfg Config) *Store {
	return &Store{
		client:           client,
		bucket:           cfg.Bucket,
		internalEndpoint: cfg.InternalEndpoint,
		publicEndpoint:   cfg.PublicEndpoint,
	}
}

func (s *Store) Bucket() string { return s.bucket }

func (s *Store) Upload(ctx context.Context, bucket, key string, r io.Reader) error {
	if err := s.client.UploadFile(ctx, bucket, key, r); err != nil {
		return domain.Internal("uploading image to storage", err)
	}
	return nil
}

func (s *Store) ViewURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	url, err := s.client.PresignGetObject(ctx, bucket, key, ttl)
	if err != nil {
		return "", domain.Internal("presigning image "+key, err)
	}
	return s.reachable(url), nil
}

func (s *Store) Delete(ctx context.Context, bucket, key string) error {
	if err := s.client.DeleteFile(ctx, bucket, key); err != nil {
		return domain.Internal("deleting image object "+key, err)
	}
	return nil
}

// reachable rewrites a presigned URL's host so the caller can resolve it.
// Locally the API talks to storage over the Docker network under one name
// while the browser reaches it under another; in production both are unset
// and the URL is returned as minted.
func (s *Store) reachable(url string) string {
	if s.publicEndpoint == "" || s.internalEndpoint == "" {
		return url
	}
	return strings.Replace(url, s.internalEndpoint, s.publicEndpoint, 1)
}
