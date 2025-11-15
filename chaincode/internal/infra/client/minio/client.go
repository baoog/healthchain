package minio

import (
	"context"
	"fmt"
	"io"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type IMinoIOClient interface {
	PutBundle(ctx context.Context, ownerId, bundleId string, r io.Reader, size int64) (uri string, err error)
	GetBundle(ctx context.Context, uri string) (io.ReadCloser, int64, error) // for read later
}

type Client struct {
	mc     *minio.Client
	bucket string
}

type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseSSL          bool
}

func NewClient(cfg Config) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("new minio: %w", err)
	}

	// Ensure bucket exists (idempotent)
	ctx := context.Background()
	exists, err := mc.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("bucket exists: %w", err)
	}
	if !exists {
		if err := mc.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
	}

	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

func (c *Client) PutBundle(ctx context.Context, ownerId, bundleId string, r io.Reader, size int64) (string, error) {
	objectName := fmt.Sprintf("ehr/%s/%s.enc", ownerId, bundleId)

	_, err := c.mc.PutObject(ctx, c.bucket, objectName, r, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}

	// Logical URI (used in Fabric)
	uri := fmt.Sprintf("minio://%s/%s", c.bucket, objectName)
	return uri, nil
}
