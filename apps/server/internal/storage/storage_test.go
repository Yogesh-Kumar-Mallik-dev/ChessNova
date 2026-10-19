package storage

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"testing"
)

func TestStorageLocalFallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chesspasant_test_avatars_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	svc := NewStorageService(Config{
		LocalFallbackDir: tmpDir,
	})

	if svc.IsS3Enabled() {
		t.Errorf("Expected S3 to be disabled without credentials")
	}

	// 1. Test invalid file type
	ctx := context.Background()
	_, err = svc.UploadAvatar(ctx, "user123", []byte("plain text not image"), "text/plain")
	if err != ErrInvalidFileType {
		t.Errorf("Expected ErrInvalidFileType, got %v", err)
	}

	// 2. Test valid PNG file
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("Failed to encode test PNG: %v", err)
	}
	url, err := svc.UploadAvatar(ctx, "user123", buf.Bytes(), "image/png")
	if err != nil {
		t.Fatalf("UploadAvatar failed for valid PNG: %v", err)
	}

	if url == "" {
		t.Errorf("Expected non-empty URL")
	}

	// 3. Test deletion
	err = svc.DeleteAvatar(ctx, url)
	if err != nil {
		t.Errorf("DeleteAvatar returned error: %v", err)
	}
}
