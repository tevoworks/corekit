package storage

import "time"

type FileMetadata struct {
	ID             int64     `json:"id"`
	Filename       string    `json:"filename"`
	SizeBytes      int64     `json:"size_bytes"`
	MIMEType       string    `json:"mime_type"`
	StoragePath    string    `json:"storage_path"`
	ChecksumSHA256 string    `json:"checksum_sha256,omitempty"`
	UploadedBy     *int64    `json:"uploaded_by"`
	IsPublic       bool      `json:"is_public"`
	URL            string    `json:"url"`
	CreatedAt      time.Time `json:"created_at"`
}
