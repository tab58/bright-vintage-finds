package fakes

import (
	"context"
	"io"
	"time"

	"main-api/internal/app/domain"
	"main-api/internal/app/ports"
)

var _ ports.ImageStore = (*FakeImageStore)(nil)

// FakeImageStore is an in-memory ports.ImageStore. Uploaded bytes are kept so
// a test can show the whole file reached storage, and Deleted records the
// objects removed.
type FakeImageStore struct {
	Err error
	// PresignErr fails only URL generation, the one storage call that runs on
	// every read path.
	PresignErr error

	BucketName string
	Objects    map[string][]byte
	Deleted    []string
}

// NewImageStore returns an empty store writing to the named bucket.
func NewImageStore(bucket string) *FakeImageStore {
	return &FakeImageStore{BucketName: bucket, Objects: map[string][]byte{}}
}

func (s *FakeImageStore) Bucket() string { return s.BucketName }

func (s *FakeImageStore) Upload(_ context.Context, bucket, key string, r io.Reader) error {
	if s.Err != nil {
		return s.Err
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.Objects[bucket+"/"+key] = body
	return nil
}

// ViewURL returns a deterministic stand-in for a presigned URL.
func (s *FakeImageStore) ViewURL(_ context.Context, bucket, key string, ttl time.Duration) (string, error) {
	if s.Err != nil {
		return "", s.Err
	}
	if s.PresignErr != nil {
		return "", s.PresignErr
	}
	if ttl != domain.PresignTTL {
		return "", domain.Internal("unexpected presign ttl", nil)
	}
	return "https://storage.test/" + bucket + "/" + key, nil
}

func (s *FakeImageStore) Delete(_ context.Context, bucket, key string) error {
	if s.Err != nil {
		return s.Err
	}
	delete(s.Objects, bucket+"/"+key)
	s.Deleted = append(s.Deleted, bucket+"/"+key)
	return nil
}
