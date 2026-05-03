package config

import "fmt"

type S3Config struct {
	Endpoint        string `env:"S3_ENDPOINT" env-default:""`            // optional, e.g. http://minio:9000
	Region          string `env:"S3_REGION" env-default:"us-east-1"`
	Bucket          string `env:"S3_BUCKET" env-required:"true"`
	AccessKeyID     string `env:"S3_ACCESS_KEY_ID" env-required:"true"`
	SecretAccessKey string `env:"S3_SECRET_ACCESS_KEY" env-required:"true"`
	UsePathStyle    bool   `env:"S3_USE_PATH_STYLE" env-default:"true"`  // MinIO needs path-style
	UseSSL          bool   `env:"S3_USE_SSL" env-default:"true"`
}

// String returns a safe representation of S3Config with masked secret.
func (c S3Config) String() string {
	return fmt.Sprintf(
		"S3Config{Endpoint:%s Region:%s Bucket:%s UsePathStyle:%t UseSSL:%t}",
		c.Endpoint, c.Region, c.Bucket, c.UsePathStyle, c.UseSSL,
	)
}
