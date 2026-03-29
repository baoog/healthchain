package minio

import (
	"consentcc/config"
	"context"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"

	minio "github.com/minio/minio-go/v7"
)

type IMinoIOClient interface {
	PutBundle(ctx context.Context, ownerId, bundleId string, r io.Reader, size int64) (string, error)
	GetObject(ctx context.Context, objectKey string) (io.ReadCloser, error)
}

type Client struct {
	mc     *minio.Client
	bucket string
}

func NewClient(cfg *config.Config) (*Client, error) {
	mc, err := minio.New(cfg.MinIOConfig.Endpoint, &minio.Options{

		Creds:  credentials.NewStaticV4(cfg.MinIOConfig.AccessKeyID, cfg.MinIOConfig.SecretAccessKey, ""),
		Secure: cfg.MinIOConfig.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("new minio: %w", err)
	}

	// Ensure bucket exists (idempotent)
	ctx := context.Background()
	exists, err := mc.BucketExists(ctx, cfg.MinIOConfig.Bucket)
	if err != nil {
		var resp minio.ErrorResponse
		if errors.As(err, &resp) {
			fmt.Printf("MinIO error: %s (code=%s, bucket=%s, resource=%s)\n",
				resp.Message, resp.Code, resp.BucketName, resp.Resource)
		} else {
			fmt.Printf("MinIO raw error: %v\n", err)
		}
		fmt.Println(err)
		return nil, err
	}

	if !exists {
		if err := mc.MakeBucket(ctx, cfg.MinIOConfig.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("make bucket: %w", err)
		}
	}

	return &Client{mc: mc, bucket: cfg.MinIOConfig.Bucket}, nil
}

func (c *Client) PutBundle(ctx context.Context, ownerId, bundleId string, r io.Reader, size int64) (string, error) {
	objectName := fmt.Sprintf("bundles/%s/%s.enc", ownerId, bundleId)

	_, err := c.mc.PutObject(ctx, c.bucket, objectName, r, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return "", fmt.Errorf("put object: %w", err)
	}

	// Logical ObjectURI (used in Fabric)
	uri := objectName
	return uri, nil
}

func (c *Client) GetObject(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	// validate we can read
	_, err = obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, fmt.Errorf("minio stat failed: %w", err)
	}
	return obj, nil
}
