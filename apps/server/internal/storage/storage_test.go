package storage

import (
	"context"
	"os"
	"testing"
)

func TestStorageLocalFallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "chessnova_test_avatars_*")
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

	// 2. Test valid PNG file (header: 89 50 4E 47 0D 0A 1A 0A)
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}
	url, err := svc.UploadAvatar(ctx, "user123", pngBytes, "image/png")
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
