package aws_s3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func testAWSConfig() aws.Config {
	return aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
	}
}

func TestNewClientOptions(t *testing.T) {
	tests := []struct {
		name          string
		options       []ClientOption
		wantEndpoint  string
		wantPathStyle bool
	}{
		{"default", nil, "", false},
		{"custom endpoint", []ClientOption{WithBaseEndpoint("http://floci:4566")}, "http://floci:4566", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(testAWSConfig(), tt.options...).(*client)
			opts := c.client.Options()
			if tt.wantEndpoint == "" {
				if opts.BaseEndpoint != nil {
					t.Errorf("BaseEndpoint = %q, want unset", *opts.BaseEndpoint)
				}
			} else if opts.BaseEndpoint == nil || *opts.BaseEndpoint != tt.wantEndpoint {
				t.Errorf("BaseEndpoint = %v, want %q", opts.BaseEndpoint, tt.wantEndpoint)
			}
			if opts.UsePathStyle != tt.wantPathStyle {
				t.Errorf("UsePathStyle = %v, want %v", opts.UsePathStyle, tt.wantPathStyle)
			}
		})
	}
}

func TestPing(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"bucket reachable", http.StatusOK, false},
		{"bucket missing", http.StatusNotFound, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			c := NewClient(testAWSConfig(), WithBaseEndpoint(srv.URL))
			err := c.Ping(context.Background(), "uploads")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Ping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
