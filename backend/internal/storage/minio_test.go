package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestMinIOUploadDownload(t *testing.T) {
	client, err := NewMinIOClient()
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("col1,col2\n1,hello\n2,world")
	hash := "test-hash-123"
	ctx := context.Background()

	// Upload
	err = client.Upload(ctx, hash, bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	// Download and verify
	reader, err := client.Download(ctx, hash)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	result, _ := io.ReadAll(reader)
	if string(result) != string(data) {
		t.Fatalf("expected %s got %s", data, result)
	}
}
