package objectstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunshu/internal/dictconfig"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/gorm"
)

type Client struct {
	cli      *minio.Client
	bucket   string
	prefix   string
	endpoint string
}

func NewFromDB(ctx context.Context, db *gorm.DB) (*Client, error) {
	cfg := dictconfig.ResolveMinioConfig(ctx, db, dictconfig.DefaultMinioDictTypes())
	return NewFromConfig(cfg)
}

func NewFromConfig(cfg dictconfig.MinioConfig) (*Client, error) {
	if !cfg.Ready() {
		return nil, fmt.Errorf("minio config incomplete: configure minio_* in data dictionary")
	}
	endpoint := dictconfig.NormalizeMinioEndpoint(cfg.Endpoint)
	cli, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}
	return &Client{cli: cli, bucket: cfg.Bucket, prefix: cfg.Prefix, endpoint: endpoint}, nil
}

// Endpoint 返回 S3 API 地址（host:port，无 scheme）。
func (c *Client) Endpoint() string {
	if c == nil {
		return ""
	}
	return c.endpoint
}

func wrapMinioError(err error, endpoint, bucket, objectKey string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "API Requests must be made to API port") {
		return fmt.Errorf("%w（minio_endpoint 须为 S3 API 端口，通常为 9000；9001 为 Web 控制台。当前 endpoint=%s bucket=%s object=%s）", err, endpoint, bucket, objectKey)
	}
	return fmt.Errorf("minio upload failed (endpoint=%s bucket=%s object=%s): %w", endpoint, bucket, objectKey, err)
}

func (c *Client) UploadFile(ctx context.Context, objectKey, localPath, contentType string) (int64, error) {
	if c == nil || c.cli == nil {
		return 0, fmt.Errorf("minio client not initialized")
	}
	key := c.prefix + strings.TrimPrefix(objectKey, "/")
	st, err := os.Stat(localPath)
	if err != nil {
		return 0, err
	}
	f, err := os.Open(localPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	if contentType == "" {
		contentType = "application/gzip"
	}
	_, err = c.cli.PutObject(ctx, c.bucket, key, f, st.Size(), minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return 0, wrapMinioError(err, c.endpoint, c.bucket, key)
	}
	return st.Size(), nil
}

func (c *Client) Bucket() string { return c.bucket }

func (c *Client) FullKey(objectKey string) string {
	return c.prefix + strings.TrimPrefix(objectKey, "/")
}

// PresignedGetURL 生成临时下载链接。
func (c *Client) PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	key := c.FullKey(objectKey)
	u, err := c.cli.PresignedGetObject(ctx, c.bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *Client) RemoveObject(ctx context.Context, objectKey string) error {
	key := c.FullKey(objectKey)
	return c.cli.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}

func DownloadToTemp(ctx context.Context, db *gorm.DB, objectKey string) (string, error) {
	cli, err := NewFromDB(ctx, db)
	if err != nil {
		return "", err
	}
	key := cli.FullKey(objectKey)
	tmp := filepath.Join(os.TempDir(), "yunshu-minio-"+filepath.Base(key))
	if err := os.MkdirAll(filepath.Dir(tmp), 0o750); err != nil {
		return "", err
	}
	if err := cli.cli.FGetObject(ctx, cli.bucket, key, tmp, minio.GetObjectOptions{}); err != nil {
		return "", err
	}
	return tmp, nil
}

func (c *Client) PutReader(ctx context.Context, objectKey string, r io.Reader, size int64, contentType string) error {
	key := c.FullKey(objectKey)
	_, err := c.cli.PutObject(ctx, c.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}
