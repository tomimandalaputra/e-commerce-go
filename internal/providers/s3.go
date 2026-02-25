package providers

import (
	"context"
	"mime/multipart"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	appconfig "github.com/tomimandalaputra/e-commerce-go/internal/config"
)

type S3Provider struct {
	client   *s3.Client
	manager  *transfermanager.Client
	bucket   string
	endpoint string
}

func NewS3Provider(cfg *appconfig.Config) *S3Provider {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.AWS.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AWS.AccessKeyID,
			cfg.AWS.SecretAccessKey,
			"",
		)),
	)

	if err != nil {
		panic("failed to create AWS config " + err.Error())
	}

	// Configure for localstack
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.AWS.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.AWS.S3Endpoint)
			o.UsePathStyle = true
		}
	})

	return &S3Provider{
		client:   client,
		manager:  transfermanager.New(client),
		bucket:   cfg.AWS.S3Bucket,
		endpoint: cfg.AWS.S3Endpoint,
	}
}

func (p *S3Provider) UploadFile(file *multipart.FileHeader, path string) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer func() { _ = src.Close() }()

	_, err = p.manager.UploadObject(context.TODO(), &transfermanager.UploadObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(path),
		Body:   src,
	})

	if err != nil {
		return "", err
	}

	return path, nil
}

func (p *S3Provider) DeleteFile(path string) error {
	_, err := p.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(strings.TrimPrefix(path, "/")),
	})

	return err
}
