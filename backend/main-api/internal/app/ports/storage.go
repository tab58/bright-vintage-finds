package ports

import (
	"context"
	"io"
	"time"
)

// ImageStore is the object storage photos live in. It is optional: a
// deployment without storage configured has none, which is why the services
// hold a nil-able value and check before reaching for it.
type ImageStore interface {
	// Bucket is where new uploads go.
	Bucket() string

	// Upload streams an object into the bucket.
	Upload(ctx context.Context, bucket, key string, r io.Reader) error

	// ViewURL returns a short-lived GET URL for an object. The URL is
	// reachable by the caller: rewriting an internal endpoint to its public
	// form is the adapter's job.
	ViewURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error)

	// Delete removes an object.
	Delete(ctx context.Context, bucket, key string) error
}
