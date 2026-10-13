package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	ErrFileTooLarge    = errors.New("file exceeds maximum allowed size (5MB)")
	ErrInvalidFileType = errors.New("invalid file format: only JPEG, PNG, WebP, and GIF images are permitted")
)

const (
	MaxAvatarSizeBytes = 5 * 1024 * 1024 // 5MB
)

type Config struct {
	Endpoint         string
	Bucket           string
	AccessKey        string
	SecretKey        string
	Region           string
	UseSSL           bool
	PublicURL        string
	LocalFallbackDir string
}

type StorageService interface {
	UploadAvatar(ctx context.Context, userID string, fileData []byte, providedType string) (string, error)
	DeleteAvatar(ctx context.Context, avatarURL string) error
	IsS3Enabled() bool
}

type S3StorageService struct {
	cfg         Config
	minioClient *minio.Client
	s3Enabled   bool
}

func NewStorageService(cfg Config) StorageService {
	s := &S3StorageService{
		cfg: cfg,
	}

	if cfg.LocalFallbackDir == "" {
		s.cfg.LocalFallbackDir = "uploads/avatars"
	}
	_ = os.MkdirAll(s.cfg.LocalFallbackDir, 0755)

	if cfg.Endpoint != "" && cfg.AccessKey != "" && cfg.SecretKey != "" && cfg.Bucket != "" {
		client, err := minio.New(cfg.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.UseSSL,
			Region: cfg.Region,
		})
		if err != nil {
			log.Printf("[storage] Failed to initialize S3 client: %v. Falling back to local storage.", err)
		} else {
			s.minioClient = client
			s.s3Enabled = true
			log.Printf("[storage] S3-compatible storage active (endpoint: %s, bucket: %s)", cfg.Endpoint, cfg.Bucket)

			// Ensure bucket exists in background
			go func() {
				ctx := context.Background()
				exists, err := client.BucketExists(ctx, cfg.Bucket)
				if err == nil && !exists {
					_ = client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region})
				}
			}()
		}
	} else {
		log.Printf("[storage] S3 credentials not provided. Using local file storage at %s", s.cfg.LocalFallbackDir)
	}

	return s
}

func (s *S3StorageService) IsS3Enabled() bool {
	return s.s3Enabled
}

func (s *S3StorageService) UploadAvatar(ctx context.Context, userID string, fileData []byte, providedType string) (string, error) {
	if len(fileData) > MaxAvatarSizeBytes {
		return "", ErrFileTooLarge
	}

	// Sniff magic bytes for true content type
	sniffedType := http.DetectContentType(fileData)
	var ext string
	var finalContentType string

	switch {
	case strings.HasPrefix(sniffedType, "image/jpeg"):
		ext = ".jpg"
		finalContentType = "image/jpeg"
	case strings.HasPrefix(sniffedType, "image/png"):
		ext = ".png"
		finalContentType = "image/png"
	case strings.HasPrefix(sniffedType, "image/webp"):
		ext = ".webp"
		finalContentType = "image/webp"
	case strings.HasPrefix(sniffedType, "image/gif"):
		ext = ".gif"
		finalContentType = "image/gif"
	default:
		return "", ErrInvalidFileType
	}

	// Generate clean randomized filename
	rnd := make([]byte, 8)
	_, _ = rand.Read(rnd)
	filename := fmt.Sprintf("avatar_%s_%s%s", userID, hex.EncodeToString(rnd), ext)

	// If S3 is enabled, upload to bucket
	if s.s3Enabled && s.minioClient != nil {
		objectKey := fmt.Sprintf("avatars/%s", filename)
		reader := bytes.NewReader(fileData)
		_, err := s.minioClient.PutObject(ctx, s.cfg.Bucket, objectKey, reader, int64(len(fileData)), minio.PutObjectOptions{
			ContentType: finalContentType,
			UserMetadata: map[string]string{
				"uploaded-by": userID,
			},
		})
		if err != nil {
			return "", fmt.Errorf("failed to upload avatar to S3: %w", err)
		}

		if s.cfg.PublicURL != "" {
			return fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.PublicURL, "/"), objectKey), nil
		}

		scheme := "https"
		if !s.cfg.UseSSL {
			scheme = "http"
		}
		return fmt.Sprintf("%s://%s/%s/%s", scheme, s.cfg.Endpoint, s.cfg.Bucket, objectKey), nil
	}

	// Fallback: Local storage
	localPath := filepath.Join(s.cfg.LocalFallbackDir, filename)
	if err := os.WriteFile(localPath, fileData, 0644); err != nil {
		return "", fmt.Errorf("failed to save avatar locally: %w", err)
	}

	return fmt.Sprintf("/uploads/avatars/%s", filename), nil
}

func (s *S3StorageService) DeleteAvatar(ctx context.Context, avatarURL string) error {
	if avatarURL == "" {
		return nil
	}

	if s.s3Enabled && s.minioClient != nil {
		// Extract object key
		parts := strings.Split(avatarURL, "/")
		if len(parts) > 0 {
			objectKey := fmt.Sprintf("avatars/%s", parts[len(parts)-1])
			_ = s.minioClient.RemoveObject(ctx, s.cfg.Bucket, objectKey, minio.RemoveObjectOptions{})
		}
		return nil
	}

	// Local deletion
	filename := filepath.Base(avatarURL)
	localPath := filepath.Join(s.cfg.LocalFallbackDir, filename)
	_ = os.Remove(localPath)
	return nil
}
