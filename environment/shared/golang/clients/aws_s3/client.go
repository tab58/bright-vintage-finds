package aws_s3

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

//go:generate mockgen -source=client.go -destination=mocks/mock_s3_client.go -package=mocks

type Client interface {
	UploadFile(ctx context.Context, bucket, key string, reader io.Reader) error
	UploadFileWithMetadata(ctx context.Context, bucket, key string, reader io.Reader, metadata map[string]string) error
	DownloadFile(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	DeleteFile(ctx context.Context, bucket, key string) error
	FileExists(ctx context.Context, bucket, key string) (bool, error)
	Ping(ctx context.Context, bucket string) error
}

type clientOptions struct {
	BaseEndpoint string
}

type ClientOption func(*clientOptions)

func WithBaseEndpoint(endpoint string) ClientOption {
	return func(o *clientOptions) {
		o.BaseEndpoint = endpoint
	}
}

type client struct {
	client *s3.Client
}

func NewClient(awsConfig aws.Config, options ...ClientOption) Client {
	// build client options
	opts := clientOptions{}
	for _, applyOption := range options {
		applyOption(&opts)
	}

	// build S3 options
	s3Options := []func(*s3.Options){}
	if opts.BaseEndpoint != "" {
		s3Options = append(s3Options, func(o *s3.Options) {
			o.UsePathStyle = true
			o.BaseEndpoint = aws.String(opts.BaseEndpoint)
		})
	}

	s3Client := s3.NewFromConfig(awsConfig, s3Options...)
	return &client{client: s3Client}
}

// Ping verifies the bucket is reachable with the configured credentials, so
// misconfiguration can surface at service boot rather than at first use.
func (c *client) Ping(ctx context.Context, bucket string) error {
	if _, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)}); err != nil {
		return fmt.Errorf("failed to reach bucket %q: %w", bucket, err)
	}
	return nil
}

func (c *client) ensureBucketExists(ctx context.Context, bucket string) error {
	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			_, createErr := c.client.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket: aws.String(bucket),
			})
			if createErr != nil {
				return fmt.Errorf("failed to create bucket: %w", createErr)
			}
			return nil
		}
		return fmt.Errorf("failed to check bucket: %w", err)
	}
	return nil
}

func (c *client) UploadFile(ctx context.Context, bucket, key string, reader io.Reader) error {
	if err := c.ensureBucketExists(ctx, bucket); err != nil {
		return err
	}

	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   reader,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}
	return nil
}

func (c *client) UploadFileWithMetadata(ctx context.Context, bucket, key string, reader io.Reader, metadata map[string]string) error {
	if err := c.ensureBucketExists(ctx, bucket); err != nil {
		return err
	}

	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		Body:     reader,
		Metadata: metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}
	return nil
}

func (c *client) DeleteFile(ctx context.Context, bucket, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}
	return nil
}

func (c *client) FileExists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return false, nil
		}
		return false, fmt.Errorf("failed to head object in S3: %w", err)
	}
	return true, nil
}

func (c *client) DownloadFile(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	resp, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}
	return resp.Body, nil
}
