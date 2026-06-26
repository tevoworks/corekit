package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	Create(ctx context.Context, f *FileMetadata) error
	GetByID(ctx context.Context, id int64) (*FileMetadata, error)
	GetPublicByID(ctx context.Context, id int64) (*FileMetadata, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit int, cursor int64, actorID *int64) ([]FileMetadata, error)
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

func (r *pgRepository) Create(ctx context.Context, f *FileMetadata) error {
	query := `
		INSERT INTO file_metadata (filename, size_bytes, mime_type, storage_path, checksum_sha256, uploaded_by, is_public)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx, query,
		f.Filename,
		f.SizeBytes,
		f.MIMEType,
		f.StoragePath,
		f.ChecksumSHA256,
		f.UploadedBy,
		f.IsPublic,
	).Scan(&f.ID, &f.CreatedAt)
}

func (r *pgRepository) GetByID(ctx context.Context, id int64) (*FileMetadata, error) {
	query := `
		SELECT id, filename, size_bytes, mime_type, storage_path, checksum_sha256, uploaded_by, is_public, created_at
		FROM file_metadata
		WHERE id = $1`

	var f FileMetadata
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, id).Scan(
		&f.ID,
		&f.Filename,
		&f.SizeBytes,
		&f.MIMEType,
		&f.StoragePath,
		&f.ChecksumSHA256,
		&f.UploadedBy,
		&f.IsPublic,
		&f.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *pgRepository) GetPublicByID(ctx context.Context, id int64) (*FileMetadata, error) {
	query := `
		SELECT id, filename, size_bytes, mime_type, storage_path, checksum_sha256, uploaded_by, is_public, created_at
		FROM file_metadata
		WHERE id = $1 AND is_public = true`

	var f FileMetadata
	q := database.GetQueryer(ctx, r.db)
	err := q.QueryRowContext(ctx, query, id).Scan(
		&f.ID,
		&f.Filename,
		&f.SizeBytes,
		&f.MIMEType,
		&f.StoragePath,
		&f.ChecksumSHA256,
		&f.UploadedBy,
		&f.IsPublic,
		&f.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *pgRepository) Delete(ctx context.Context, id int64) error {
	query := `
		DELETE FROM file_metadata
		WHERE id = $1`
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, query, id)
	return err
}

func (r *pgRepository) List(ctx context.Context, limit int, cursor int64, actorID *int64) ([]FileMetadata, error) {
	query := `
		SELECT id, filename, size_bytes, mime_type, storage_path, checksum_sha256, uploaded_by, is_public, created_at
		FROM file_metadata`
	args := []interface{}{}
	argIdx := 0

	if actorID != nil {
		argIdx++
		query += fmt.Sprintf(` WHERE uploaded_by = $%d`, argIdx)
		args = append(args, *actorID)
	}

	if cursor > 0 {
		argIdx++
		if actorID != nil {
			query += fmt.Sprintf(` AND id < $%d`, argIdx)
		} else {
			query += fmt.Sprintf(` WHERE id < $%d`, argIdx)
		}
		args = append(args, cursor)
	}

	argIdx++
	query += fmt.Sprintf(` ORDER BY id DESC LIMIT $%d`, argIdx)
	args = append(args, limit)

	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []FileMetadata
	for rows.Next() {
		var f FileMetadata
		err := rows.Scan(
			&f.ID,
			&f.Filename,
			&f.SizeBytes,
			&f.MIMEType,
			&f.StoragePath,
			&f.ChecksumSHA256,
			&f.UploadedBy,
			&f.IsPublic,
			&f.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}
