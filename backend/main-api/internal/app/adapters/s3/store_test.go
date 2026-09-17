package s3

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"main-api/internal/app/domain"

	aws_s3 "github.com/tab58/bright-vintage-finds/environment/shared/golang/clients/aws_s3"
)

// stubClient implements only what the store calls; the embedded interface
// leaves the rest as nil methods that panic if they are ever reached.
type stubClient struct {
	aws_s3.Client

	url      string
	err      error
	uploaded map[string]string
}

func (c stubClient) PresignGetObject(context.Context, string, string, time.Duration) (string, error) {
	return c.url, c.err
}

func (c stubClient) UploadFile(_ context.Context, bucket, key string, r io.Reader) error {
	if c.err != nil {
		return c.err
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	c.uploaded[bucket+"/"+key] = string(body)
	return nil
}

func (c stubClient) DeleteFile(_ context.Context, bucket, key string) error {
	if c.err != nil {
		return c.err
	}
	c.uploaded[bucket+"/"+key] = ""
	return nil
}

func TestViewURLRewritesTheEndpoint(t *testing.T) {
	tests := []struct {
		name   string
		cfg    Config
		minted string
		want   string
	}{
		{
			name:   "local development swaps the internal host for the public one",
			cfg:    Config{InternalEndpoint: "http://floci:9000", PublicEndpoint: "http://localhost:9000"},
			minted: "http://floci:9000/bucket/items/i1/a.jpg?X-Amz-Signature=abc",
			want:   "http://localhost:9000/bucket/items/i1/a.jpg?X-Amz-Signature=abc",
		},
		{
			name:   "production leaves the URL as minted",
			cfg:    Config{InternalEndpoint: "https://storage.example"},
			minted: "https://storage.example/bucket/key?X-Amz-Signature=abc",
			want:   "https://storage.example/bucket/key?X-Amz-Signature=abc",
		},
		{
			name:   "a public endpoint with no internal one to match is ignored",
			cfg:    Config{PublicEndpoint: "http://localhost:9000"},
			minted: "http://floci:9000/bucket/key",
			want:   "http://floci:9000/bucket/key",
		},
		{
			name:   "only the host is rewritten, not a later occurrence",
			cfg:    Config{InternalEndpoint: "http://floci:9000", PublicEndpoint: "http://localhost:9000"},
			minted: "http://floci:9000/bucket/key?next=http://floci:9000/other",
			want:   "http://localhost:9000/bucket/key?next=http://floci:9000/other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := New(stubClient{url: tt.minted}, tt.cfg)

			got, err := store.ViewURL(context.Background(), "bucket", "key", domain.PresignTTL)
			if err != nil {
				t.Fatalf("ViewURL: %v", err)
			}
			if got != tt.want {
				t.Errorf("ViewURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestViewURLReportsPresignFailureAsInternal(t *testing.T) {
	cause := errors.New("credentials expired")
	store := New(stubClient{err: cause}, Config{Bucket: "b"})

	_, err := store.ViewURL(context.Background(), "b", "items/i1/a.jpg", domain.PresignTTL)
	if domain.KindOf(err) != domain.KindInternal {
		t.Errorf("kind = %d, want internal", domain.KindOf(err))
	}
	if !errors.Is(err, cause) {
		t.Errorf("err = %v, should unwrap to the cause", err)
	}
}

func TestUploadAndDeletePassThroughAndWrapFailures(t *testing.T) {
	ctx := context.Background()
	client := stubClient{uploaded: map[string]string{}}
	store := New(client, Config{Bucket: "uploads"})

	if err := store.Upload(ctx, "uploads", "items/i1/a.jpg", strings.NewReader("bytes")); err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if got := client.uploaded["uploads/items/i1/a.jpg"]; got != "bytes" {
		t.Errorf("stored %q, want the whole body", got)
	}
	if err := store.Delete(ctx, "uploads", "items/i1/a.jpg"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	cause := errors.New("bucket is gone")
	broken := New(stubClient{err: cause, uploaded: map[string]string{}}, Config{Bucket: "uploads"})

	uploadErr := broken.Upload(ctx, "uploads", "k", strings.NewReader("x"))
	if !errors.Is(uploadErr, cause) || domain.KindOf(uploadErr) != domain.KindInternal {
		t.Errorf("upload err = %v (kind %d), want the cause as internal", uploadErr, domain.KindOf(uploadErr))
	}
	deleteErr := broken.Delete(ctx, "uploads", "k")
	if !errors.Is(deleteErr, cause) || domain.KindOf(deleteErr) != domain.KindInternal {
		t.Errorf("delete err = %v (kind %d), want the cause as internal", deleteErr, domain.KindOf(deleteErr))
	}
}

func TestBucketIsTheConfiguredUploadTarget(t *testing.T) {
	store := New(stubClient{}, Config{Bucket: "uploads"})
	if store.Bucket() != "uploads" {
		t.Errorf("Bucket() = %q, want uploads", store.Bucket())
	}
}
