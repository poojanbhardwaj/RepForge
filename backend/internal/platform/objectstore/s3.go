package objectstore

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Store interface {
	Put(context.Context, string, io.Reader) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}

type S3 struct {
	client *s3.Client
	bucket string
}

type Options struct {
	Endpoint       string
	Region         string
	AccessKey      string
	SecretKey      string
	Bucket         string
	ForcePathStyle bool
}

func NewS3(ctx context.Context, options Options) (*S3, error) {
	awsConfig, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(options.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(options.AccessKey, options.SecretKey, "")),
	)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsConfig, func(value *s3.Options) {
		value.BaseEndpoint = aws.String(options.Endpoint)
		value.UsePathStyle = options.ForcePathStyle
	})
	return &S3{client: client, bucket: options.Bucket}, nil
}

func (s *S3) Put(ctx context.Context, key string, body io.Reader) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key), Body: body})
	return err
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return err
}
