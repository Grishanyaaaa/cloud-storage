package objectstorage

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
	infraconfig "github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/config"
)

// S3Client implements the parts of port.ObjectStorage that are NOT pre-signing.
// Pre-signing is in s3_presigner.go because it uses a separate s3.PresignClient.
type S3Client struct {
	client     *s3.Client
	presigner  *s3.PresignClient
	bucket     string
}

// NewS3Client builds an aws-sdk-go-v2 S3 client compatible with both AWS S3 and MinIO.
func NewS3Client(ctx context.Context, cfg infraconfig.S3Config) (*S3Client, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("s3: bucket is required")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("aws load config: %w", err)
	}

	cli := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
		o.UsePathStyle = cfg.UsePathStyle
	})
	pre := s3.NewPresignClient(cli)
	return &S3Client{client: cli, presigner: pre, bucket: cfg.Bucket}, nil
}

// HeadObject returns metadata of an existing object or nil when missing.
func (c *S3Client) HeadObject(ctx context.Context, key valueobject.StorageKey) (*port.ObjectMetadata, error) {
	out, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key.String()),
	})
	if err != nil {
		if isNotFoundError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("head object: %w", err)
	}
	meta := &port.ObjectMetadata{}
	if out.ContentLength != nil {
		meta.SizeBytes = *out.ContentLength
	}
	if out.ETag != nil {
		meta.ETag = *out.ETag
	}
	if out.ContentType != nil {
		meta.MimeType = *out.ContentType
	}
	return meta, nil
}

// DeleteObject removes an object. Idempotent (no error on missing).
func (c *S3Client) DeleteObject(ctx context.Context, key valueobject.StorageKey) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key.String()),
	})
	if err != nil && !isNotFoundError(err) {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

// isNotFoundError reports whether err signals a NoSuchKey / 404.
func isNotFoundError(err error) bool {
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var nf *types.NotFound
	if errors.As(err, &nf) {
		return true
	}
	var ae smithy.APIError
	if errors.As(err, &ae) {
		switch ae.ErrorCode() {
		case "NoSuchKey", "NotFound", "404":
			return true
		}
	}
	var rerr interface {
		HTTPStatusCode() int
	}
	if errors.As(err, &rerr) && rerr.HTTPStatusCode() == http.StatusNotFound {
		return true
	}
	return false
}
