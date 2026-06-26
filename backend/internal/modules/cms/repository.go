package cms

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/lib/pq"
	"github.com/tevoworks/corekit/backend/internal/database"
)

type Repository interface {
	CreatePage(ctx context.Context, p *Page) error
	GetPageByID(ctx context.Context, id int64) (*Page, error)
	GetPageBySlug(ctx context.Context, slug string) (*Page, error)
	ListPages(ctx context.Context, limit int, cursor int64, status string) ([]Page, error)
	ListPublishedPages(ctx context.Context, limit int, cursor int64) ([]Page, error)
	UpdatePage(ctx context.Context, p *Page) error
	DeletePage(ctx context.Context, id int64) error
	PublishPage(ctx context.Context, id int64) error
	UnpublishPage(ctx context.Context, id int64) error
	CheckSlugExists(ctx context.Context, slug string, excludeID int64) (bool, error)

	CreatePost(ctx context.Context, p *BlogPost) error
	GetPostByID(ctx context.Context, id int64) (*BlogPost, error)
	GetPostBySlug(ctx context.Context, slug string) (*BlogPost, error)
	ListPosts(ctx context.Context, limit int, cursor int64, status string) ([]BlogPost, error)
	ListPublishedPosts(ctx context.Context, limit int, cursor int64) ([]BlogPost, error)
	UpdatePost(ctx context.Context, p *BlogPost) error
	DeletePost(ctx context.Context, id int64) error
	PublishPost(ctx context.Context, id int64) error
	UnpublishPost(ctx context.Context, id int64) error
	IncrementPostViewCount(ctx context.Context, id int64) error

	CreateSection(ctx context.Context, s *PageSection) error
	ListSectionsByPageID(ctx context.Context, pageID int64) ([]PageSection, error)
	GetSectionByID(ctx context.Context, id int64) (*PageSection, error)
	UpdateSection(ctx context.Context, s *PageSection) error
	DeleteSection(ctx context.Context, id int64) error
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

const pageColumns = `id, title, slug, content, meta_title, meta_description, og_image, featured_image_id, status, published_at, created_by, created_at, updated_at, deleted_at`
const postColumns = `id, title, slug, content, excerpt, meta_title, meta_description, og_image, featured_image_id, author_id, status, published_at, tags, view_count, created_at, updated_at, deleted_at`
const sectionColumns = `id, page_id, type, title, content, sort_order, created_at, updated_at`

func scanPage(sc interface{ Scan(dest ...interface{}) error }) (Page, error) {
	var p Page
	var featuredImageID sql.NullInt64
	err := sc.Scan(
		&p.ID, &p.Title, &p.Slug, &p.Content,
		&p.MetaTitle, &p.MetaDescription, &p.OgImage,
		&featuredImageID,
		&p.Status, &p.PublishedAt, &p.CreatedBy,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	)
	if err != nil {
		return p, err
	}
	if featuredImageID.Valid {
		p.FeaturedImageID = &featuredImageID.Int64
	}
	return p, nil
}

func scanPost(sc interface{ Scan(dest ...interface{}) error }) (BlogPost, error) {
	var bp BlogPost
	var featuredImageID sql.NullInt64
	err := sc.Scan(
		&bp.ID, &bp.Title, &bp.Slug, &bp.Content, &bp.Excerpt,
		&bp.MetaTitle, &bp.MetaDescription, &bp.OgImage,
		&featuredImageID,
		&bp.AuthorID, &bp.Status, &bp.PublishedAt,
		pq.Array(&bp.Tags), &bp.ViewCount,
		&bp.CreatedAt, &bp.UpdatedAt, &bp.DeletedAt,
	)
	if err != nil {
		return bp, err
	}
	if featuredImageID.Valid {
		bp.FeaturedImageID = &featuredImageID.Int64
	}
	if bp.Tags == nil {
		bp.Tags = []string{}
	}
	return bp, nil
}

func scanSection(sc interface{ Scan(dest ...interface{}) error }) (PageSection, error) {
	var ps PageSection
	var contentBytes []byte
	err := sc.Scan(
		&ps.ID, &ps.PageID, &ps.Type, &ps.Title,
		&contentBytes, &ps.SortOrder, &ps.CreatedAt, &ps.UpdatedAt,
	)
	if err != nil {
		return ps, err
	}
	if contentBytes != nil {
		ps.Content = json.RawMessage(contentBytes)
	}
	return ps, nil
}

// --- Pages ---

func (r *pgRepository) CreatePage(ctx context.Context, p *Page) error {
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx,
		`INSERT INTO cms_pages (title, slug, content, meta_title, meta_description, og_image, featured_image_id, status, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at, updated_at`,
		p.Title, p.Slug, p.Content, p.MetaTitle, p.MetaDescription, p.OgImage, p.FeaturedImageID, p.Status, p.CreatedBy,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *pgRepository) GetPageByID(ctx context.Context, id int64) (*Page, error) {
	q := database.GetQueryer(ctx, r.db)
	p, err := scanPage(q.QueryRowContext(ctx,
		`SELECT `+pageColumns+` FROM cms_pages WHERE id = $1 AND deleted_at IS NULL`, id,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *pgRepository) GetPageBySlug(ctx context.Context, slug string) (*Page, error) {
	q := database.GetQueryer(ctx, r.db)
	p, err := scanPage(q.QueryRowContext(ctx,
		`SELECT `+pageColumns+` FROM cms_pages WHERE slug = $1 AND deleted_at IS NULL`, slug,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *pgRepository) ListPages(ctx context.Context, limit int, cursor int64, status string) ([]Page, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT `+pageColumns+` FROM cms_pages WHERE id > $2 AND deleted_at IS NULL AND ($3 = '' OR status = $3) ORDER BY id ASC LIMIT $1`,
		limit, cursor, status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []Page
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (r *pgRepository) ListPublishedPages(ctx context.Context, limit int, cursor int64) ([]Page, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT `+pageColumns+` FROM cms_pages WHERE id > $2 AND deleted_at IS NULL AND status = 'published' ORDER BY id ASC LIMIT $1`,
		limit, cursor,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pages []Page
	for rows.Next() {
		p, err := scanPage(rows)
		if err != nil {
			return nil, err
		}
		pages = append(pages, p)
	}
	return pages, rows.Err()
}

func (r *pgRepository) UpdatePage(ctx context.Context, p *Page) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_pages SET title=$1, slug=$2, content=$3, meta_title=$4, meta_description=$5, og_image=$6, featured_image_id=$7, updated_at=NOW() WHERE id=$8 AND deleted_at IS NULL`,
		p.Title, p.Slug, p.Content, p.MetaTitle, p.MetaDescription, p.OgImage, p.FeaturedImageID, p.ID,
	)
	return err
}

func (r *pgRepository) DeletePage(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_pages SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) PublishPage(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_pages SET status='published', published_at=COALESCE(published_at, NOW()), updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) UnpublishPage(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_pages SET status='draft', updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) CheckSlugExists(ctx context.Context, slug string, excludeID int64) (bool, error) {
	q := database.GetQueryer(ctx, r.db)
	var exists bool
	err := q.QueryRowContext(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM cms_pages WHERE slug = $1 AND deleted_at IS NULL AND id != $2
			UNION ALL
			SELECT 1 FROM cms_blog_posts WHERE slug = $1 AND deleted_at IS NULL AND id != $2
		)`,
		slug, excludeID,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// --- Blog Posts ---

func (r *pgRepository) CreatePost(ctx context.Context, p *BlogPost) error {
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx,
		`INSERT INTO cms_blog_posts (title, slug, content, excerpt, meta_title, meta_description, og_image, featured_image_id, author_id, status, tags)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, created_at, updated_at`,
		p.Title, p.Slug, p.Content, p.Excerpt, p.MetaTitle, p.MetaDescription, p.OgImage, p.FeaturedImageID, p.AuthorID, p.Status, pq.Array(p.Tags),
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

func (r *pgRepository) GetPostByID(ctx context.Context, id int64) (*BlogPost, error) {
	q := database.GetQueryer(ctx, r.db)
	bp, err := scanPost(q.QueryRowContext(ctx,
		`SELECT `+postColumns+` FROM cms_blog_posts WHERE id = $1 AND deleted_at IS NULL`, id,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &bp, nil
}

func (r *pgRepository) GetPostBySlug(ctx context.Context, slug string) (*BlogPost, error) {
	q := database.GetQueryer(ctx, r.db)
	bp, err := scanPost(q.QueryRowContext(ctx,
		`SELECT `+postColumns+` FROM cms_blog_posts WHERE slug = $1 AND deleted_at IS NULL`, slug,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &bp, nil
}

func (r *pgRepository) ListPosts(ctx context.Context, limit int, cursor int64, status string) ([]BlogPost, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT `+postColumns+` FROM cms_blog_posts WHERE id > $2 AND deleted_at IS NULL AND ($3 = '' OR status = $3) ORDER BY id ASC LIMIT $1`,
		limit, cursor, status,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []BlogPost
	for rows.Next() {
		bp, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, bp)
	}
	return posts, rows.Err()
}

func (r *pgRepository) ListPublishedPosts(ctx context.Context, limit int, cursor int64) ([]BlogPost, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT `+postColumns+` FROM cms_blog_posts WHERE id > $2 AND deleted_at IS NULL AND status = 'published' ORDER BY id ASC LIMIT $1`,
		limit, cursor,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []BlogPost
	for rows.Next() {
		bp, err := scanPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, bp)
	}
	return posts, rows.Err()
}

func (r *pgRepository) UpdatePost(ctx context.Context, p *BlogPost) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_blog_posts SET title=$1, slug=$2, content=$3, excerpt=$4, meta_title=$5, meta_description=$6, og_image=$7, featured_image_id=$8, tags=$9, updated_at=NOW() WHERE id=$10 AND deleted_at IS NULL`,
		p.Title, p.Slug, p.Content, p.Excerpt, p.MetaTitle, p.MetaDescription, p.OgImage, p.FeaturedImageID, pq.Array(p.Tags), p.ID,
	)
	return err
}

func (r *pgRepository) DeletePost(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_blog_posts SET deleted_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) PublishPost(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_blog_posts SET status='published', published_at=COALESCE(published_at, NOW()), updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) UnpublishPost(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_blog_posts SET status='draft', updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) IncrementPostViewCount(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_blog_posts SET view_count = view_count + 1 WHERE id = $1`, id,
	)
	return err
}

// --- Page Sections ---

func (r *pgRepository) CreateSection(ctx context.Context, s *PageSection) error {
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx,
		`INSERT INTO cms_page_sections (page_id, type, title, content, sort_order)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		s.PageID, s.Type, s.Title, []byte(s.Content), s.SortOrder,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

func (r *pgRepository) ListSectionsByPageID(ctx context.Context, pageID int64) ([]PageSection, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT `+sectionColumns+` FROM cms_page_sections WHERE page_id = $1 ORDER BY sort_order ASC`, pageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sections []PageSection
	for rows.Next() {
		ps, err := scanSection(rows)
		if err != nil {
			return nil, err
		}
		sections = append(sections, ps)
	}
	return sections, rows.Err()
}

func (r *pgRepository) GetSectionByID(ctx context.Context, id int64) (*PageSection, error) {
	q := database.GetQueryer(ctx, r.db)
	ps, err := scanSection(q.QueryRowContext(ctx,
		`SELECT `+sectionColumns+` FROM cms_page_sections WHERE id = $1`, id,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &ps, nil
}

func (r *pgRepository) UpdateSection(ctx context.Context, s *PageSection) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_page_sections SET type=$1, title=$2, content=$3, sort_order=$4, updated_at=NOW() WHERE id=$5`,
		s.Type, s.Title, []byte(s.Content), s.SortOrder, s.ID,
	)
	return err
}

func (r *pgRepository) DeleteSection(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, `DELETE FROM cms_page_sections WHERE id = $1`, id)
	return err
}
