package objectstorage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
)

// Compile-time check: S3Client implements port.ObjectStorage.
var _ port.ObjectStorage = (*S3Client)(nil)

// PresignUpload generates a single pre-signed PUT URL for direct upload to S3.
// Content-Length is NOT signed because browsers block it as an unsafe header.
// The browser sets Content-Length automatically when sending the request body.
func (c *S3Client) PresignUpload(ctx context.Context, in port.PresignUploadInput) (*port.PresignedURL, error) {
	now := time.Now()
	put, err := c.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(in.Key.String()),
		ContentType: aws.String(in.MimeType.String()),
		// ContentLength is intentionally omitted from presigning to avoid signature mismatch.
		// Browsers automatically set Content-Length but block manual setting via XHR.
	}, s3.WithPresignExpires(in.TTL))
	if err != nil {
		return nil, fmt.Errorf("presign put: %w", err)
	}
	headers := map[string]string{
		"Content-Type": in.MimeType.String(),
		// Content-Length is omitted — browser sets it automatically
	}
	return &port.PresignedURL{
		URL:       put.URL,
		Method:    put.Method,
		Headers:   headers,
		ExpiresAt: now.Add(in.TTL),
	}, nil
}

// PresignDownload generates a single pre-signed GET URL for direct download from S3.
// The Content-Disposition header is signed into the URL so the browser respects it.
func (c *S3Client) PresignDownload(ctx context.Context, in port.PresignDownloadInput) (*port.PresignedURL, error) {
	now := time.Now()
	disposition := in.Disposition
	if disposition == "" {
		disposition = "attachment"
	}
	cd := disposition
	if in.Filename != "" {
		cd = fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`,
			disposition, sanitizeForHeader(in.Filename), url.PathEscape(in.Filename),
		)
	}
	input := &s3.GetObjectInput{
		Bucket:                     aws.String(c.bucket),
		Key:                        aws.String(in.Key.String()),
		ResponseContentDisposition: aws.String(cd),
	}
	if in.ResponseMimeType != "" {
		input.ResponseContentType = aws.String(in.ResponseMimeType)
	}
	get, err := c.presigner.PresignGetObject(ctx, input, s3.WithPresignExpires(in.TTL))
	if err != nil {
		return nil, fmt.Errorf("presign get: %w", err)
	}
	return &port.PresignedURL{
		URL:       get.URL,
		Method:    get.Method,
		Headers:   nil,
		ExpiresAt: now.Add(in.TTL),
	}, nil
}

// sanitizeForHeader replaces characters that would break a Content-Disposition header.
func sanitizeForHeader(s string) string {
	r := strings.NewReplacer(
		`"`, `'`,
		"\r", "",
		"\n", "",
	)
	return r.Replace(s)
}
