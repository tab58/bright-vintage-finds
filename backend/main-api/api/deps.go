package api

import (
	db_platform "main-api/db"

	aws_s3 "github.com/tab58/bright-vintage-finds/environment/shared/golang/clients/aws_s3"
)

// AppDeps carries the application-level dependencies the API handlers need.
// It is optional: with nil, only platform routes (/healthz) are registered.
type AppDeps struct {
	DB    *db_platform.Client
	Store aws_s3.Client
	// S3UploadBucket is the bucket images upload to; required when Store is set.
	S3UploadBucket string
	// S3PublicEndpoint, when set, rewrites presigned URL hosts so they resolve
	// outside the storage's internal network (local dev only).
	S3PublicEndpoint string
	// S3InternalEndpoint is the internal base (S3_BASE_ENDPOINT) that presigned
	// URLs carry before the rewrite above.
	S3InternalEndpoint string
}

func (d *AppDeps) hasDB() bool { return d != nil && d.DB != nil }