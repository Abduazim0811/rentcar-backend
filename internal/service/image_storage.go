package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const maxImageSize = 5 << 20

type ImageStorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
	PublicURL string
}

type UploadedImage struct {
	ObjectName  string `json:"object_name"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type ImageStorage struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func NewImageStorage(ctx context.Context, cfg ImageStorageConfig) (*ImageStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}

	storage := &ImageStorage{client: client, bucket: cfg.Bucket, publicURL: strings.TrimRight(cfg.PublicURL, "/")}
	if err := storage.ensureBucket(ctx); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *ImageStorage) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*UploadedImage, error) {
	if header.Size <= 0 || header.Size > maxImageSize {
		return nil, fmt.Errorf("image size must be between 1 byte and 5 MB")
	}

	contentType := header.Header.Get("Content-Type")
	if !isAllowedImage(contentType) {
		return nil, fmt.Errorf("only jpeg, png, webp, and gif images are allowed")
	}

	objectName := fmt.Sprintf("%s-%s%s", time.Now().UTC().Format("20060102150405"), randomHex(8), imageExt(header.Filename, contentType))
	_, err := s.client.PutObject(ctx, s.bucket, objectName, file, header.Size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, err
	}

	return &UploadedImage{
		ObjectName:  objectName,
		URL:         s.publicURL + "/" + objectName,
		ContentType: contentType,
		Size:        header.Size,
	}, nil
}

func (s *ImageStorage) Get(ctx context.Context, objectName string) (io.ReadCloser, minio.ObjectInfo, error) {
	object, err := s.client.GetObject(ctx, s.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, minio.ObjectInfo{}, err
	}

	info, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, minio.ObjectInfo{}, err
	}

	return object, info, nil
}

func (s *ImageStorage) ensureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
}

func isAllowedImage(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}

func imageExt(filename, contentType string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return ext
	}

	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".img"
	}
}

func randomHex(bytesCount int) string {
	buf := make([]byte, bytesCount)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
