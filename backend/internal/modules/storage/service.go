package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
)

type StorageProvider interface {
	Upload(ctx context.Context, key string, size int64, content io.Reader) (string, error)
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type S3StorageProvider struct {
	client *s3.Client
	bucket string
}

func NewS3StorageProvider(endpoint, region, bucket, accessKey, secretKey string) (*S3StorageProvider, error) {
	if bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET must be set")
	}

	var opts []func(*config.LoadOptions) error
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	} else {
		opts = append(opts, config.WithRegion("us-east-1"))
	}

	if accessKey != "" && secretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	}

	// Disable trailing checksum for non-TLS endpoints (e.g., local MinIO)
	opts = append(opts, config.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired))

	cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})

	return &S3StorageProvider{
		client: client,
		bucket: bucket,
	}, nil
}

func (p *S3StorageProvider) Upload(ctx context.Context, key string, size int64, content io.Reader) (string, error) {
	_, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(p.bucket),
		Key:           aws.String(key),
		Body:          content,
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return "", err
	}
	return key, nil
}

func (p *S3StorageProvider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (p *S3StorageProvider) Delete(ctx context.Context, key string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	})
	return err
}

type Service interface {
	UploadFile(ctx context.Context, filename string, size int64, mimeType string, content io.Reader, actorID int64, isPublic bool) (*FileMetadata, error)
	DownloadFile(ctx context.Context, id int64, actorID int64, isSuperAdmin bool) (*FileMetadata, io.ReadCloser, error)
	DownloadPublicFile(ctx context.Context, id int64) (*FileMetadata, io.ReadCloser, error)
	DeleteFile(ctx context.Context, id int64, actorID int64) error
	ListFiles(ctx context.Context, limit int, cursor int64, actorID int64, isSuperAdmin bool) ([]FileMetadata, error)
}

type service struct {
	db           *sql.DB
	repo         Repository
	provider     StorageProvider
	auditService audit.Service
}

func NewService(db *sql.DB, repo Repository, provider StorageProvider, auditService audit.Service) Service {
	return &service{
		db:           db,
		repo:         repo,
		provider:     provider,
		auditService: auditService,
	}
}

func (s *service) UploadFile(ctx context.Context, filename string, size int64, mimeType string, content io.Reader, actorID int64, isPublic bool) (*FileMetadata, error) {
	var actorPtr *int64
	if actorID > 0 {
		actorPtr = &actorID
	}

	sanitizedName := filepath.Base(filename)
	uniqueFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), sanitizedName)

	// Read entire content into buffer so S3 SDK can seek for hash computation
	buf, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("read content: %w", err)
	}

	hasher := sha256.New()
	if _, err := hasher.Write(buf); err != nil {
		return nil, fmt.Errorf("compute hash: %w", err)
	}

	storagePath, err := s.provider.Upload(ctx, uniqueFilename, int64(len(buf)), bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	f := &FileMetadata{
		Filename:       filename,
		SizeBytes:      size,
		MIMEType:       mimeType,
		StoragePath:    storagePath,
		ChecksumSHA256: hex.EncodeToString(hasher.Sum(nil)),
		UploadedBy:     actorPtr,
		IsPublic:       isPublic,
	}

	actx := database.WithAuditCtx(ctx, actorID, "UPLOAD_FILE")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Create(txCtx, f)
	})

	if err != nil {
		if delErr := s.provider.Delete(ctx, uniqueFilename); delErr != nil {
			slog.Error("upload: DB transaction failed and rollback deletion also failed", "original_error", err, "rollback_error", delErr)
		}
		return nil, err
	}

	return f, nil
}

func (s *service) DownloadFile(ctx context.Context, id int64, actorID int64, isSuperAdmin bool) (*FileMetadata, io.ReadCloser, error) {
	meta, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if meta == nil {
		return nil, nil, database.ErrNotFound
	}

	if !isSuperAdmin && !meta.IsPublic && (meta.UploadedBy == nil || *meta.UploadedBy != actorID) {
		return nil, nil, fmt.Errorf("access denied: file is private")
	}

	key := filepath.Base(meta.StoragePath)
	file, err := s.provider.Download(ctx, key)
	if err != nil {
		return nil, nil, err
	}

	return meta, file, nil
}

func (s *service) DownloadPublicFile(ctx context.Context, id int64) (*FileMetadata, io.ReadCloser, error) {
	meta, err := s.repo.GetPublicByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if meta == nil {
		return nil, nil, database.ErrNotFound
	}
	if !meta.IsPublic {
		return nil, nil, fmt.Errorf("access denied: file is private")
	}

	key := filepath.Base(meta.StoragePath)
	file, err := s.provider.Download(ctx, key)
	if err != nil {
		return nil, nil, err
	}

	return meta, file, nil
}

func (s *service) DeleteFile(ctx context.Context, id int64, actorID int64) error {
	meta, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if meta == nil {
		return database.ErrNotFound
	}

	if meta.UploadedBy == nil || *meta.UploadedBy != actorID {
		return fmt.Errorf("access denied: you can only delete your own files")
	}

	key := filepath.Base(meta.StoragePath)
	if err := s.provider.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete file from storage: %w", err)
	}

	actx := database.WithAuditCtx(ctx, actorID, "DELETE_FILE")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.Delete(txCtx, id)
	})
}

func (s *service) ListFiles(ctx context.Context, limit int, cursor int64, actorID int64, isSuperAdmin bool) ([]FileMetadata, error) {
	var actorPtr *int64
	if !isSuperAdmin && actorID > 0 {
		actorPtr = &actorID
	}
	return s.repo.List(ctx, limit, cursor, actorPtr)
}
