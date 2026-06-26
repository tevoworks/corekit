package cms

import (
	"encoding/json"
	"time"
)

type Page struct {
	ID              int64      `json:"id"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Content         string     `json:"content"`
	MetaTitle       string     `json:"meta_title"`
	MetaDescription string     `json:"meta_description"`
	OgImage         string     `json:"og_image"`
	FeaturedImageID *int64     `json:"featured_image_id,omitempty"`
	Status          string     `json:"status"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	CreatedBy       int64      `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

type BlogPost struct {
	ID              int64      `json:"id"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Content         string     `json:"content"`
	Excerpt         string     `json:"excerpt"`
	MetaTitle       string     `json:"meta_title"`
	MetaDescription string     `json:"meta_description"`
	OgImage         string     `json:"og_image"`
	FeaturedImageID *int64     `json:"featured_image_id,omitempty"`
	AuthorID        int64      `json:"author_id"`
	Status          string     `json:"status"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	Tags            []string   `json:"tags"`
	ViewCount       int        `json:"view_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

type PageSection struct {
	ID        int64           `json:"id"`
	PageID    int64           `json:"page_id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Content   json.RawMessage `json:"content"`
	SortOrder int             `json:"sort_order"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type SlugCheckResult struct {
	Available bool   `json:"available"`
	Slug      string `json:"slug"`
}
