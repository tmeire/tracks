package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tmeire/tracks/storage"
)

type Driver struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
}

var _ storage.Driver = (*Driver)(nil)

func NewDriver(client *s3.Client, bucket string) *Driver {
	return &Driver{
		client:        client,
		presignClient: s3.NewPresignClient(client),
		bucket:        bucket,
	}
}

func (d *Driver) Put(ctx context.Context, key string, r io.Reader) error {
	_, err := d.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	return err
}

func (d *Driver) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := d.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return output.Body, nil
}

func (d *Driver) Delete(ctx context.Context, key string) error {
	_, err := d.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (d *Driver) URL(ctx context.Context, key string, expires time.Duration) (string, error) {
	request, err := d.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return request.URL, nil
}

func (d *Driver) SignUpload(ctx context.Context, key string, expires time.Duration, contentType string) (string, error) {
	request, err := d.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(d.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return request.URL, nil
}
