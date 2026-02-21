package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const bucketName = "datasets"

type MinIOClient struct {
	client *minio.Client
}

func NewMinIOClient() (*MinIOClient, error) {
	client, err := minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("minioadmin", "minioadmin", ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	return &MinIOClient{client: client}, nil
}

func (m *MinIOClient) Upload(ctx context.Context, hash string, data io.Reader, size int64) error {
	_, err := m.client.PutObject(ctx, bucketName, hash, data, size, minio.PutObjectOptions{
		ContentType: "text/csv",
	})
	return err
}

func (m *MinIOClient) Download(ctx context.Context, hash string) (io.ReadCloser, error) {
	obj, err := m.client.GetObject(ctx, bucketName, hash, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}
