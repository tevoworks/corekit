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
	// Pages
	CreatePage(ctx context.Context, page *Page) error
	GetPageByID(ctx context.Context, id int64) (*Page, error)
	GetPageBySlug(ctx context.Context, slug string) (*Page, error)
	ListPages(ctx context.Context, limit int, cursor int64) ([]Page, error)
	ListPublishedPages(ctx context.Context, limit int, cursor int64) ([]Page, error)
	UpdatePage(ctx context.Context, page *Page) error
	DeletePage(ctx context.Context, id int64) error
	PublishPage(ctx context.Context, id int64) error
	UnpublishPage(ctx context.Context, id int64) error

	// Blog Posts
	CreateBlogPost(ctx context.Context, post *BlogPost) error
	GetBlogPostByID(ctx context.Context, id int64) (*BlogPost, error)
	GetBlogPostBySlug(ctx context.Context, slug string) (*BlogPost, error)
	ListBlogPosts(ctx context.Context, limit int, cursor int64) ([]BlogPost, error)
	ListPublishedBlogPosts(ctx context.Context, limit int, cursor int64) ([]BlogPost, error)
	UpdateBlogPost(ctx context.Context, post *BlogPost) error
	DeleteBlogPost(ctx context.Context, id int64) error
	PublishBlogPost(ctx context.Context, id int64) error
	UnpublishBlogPost(ctx context.Context, id int64) error
	IncrementViewCount(ctx context.Context, id int64) error

	// Page Sections
	CreatePageSection(ctx context.Context, section *PageSection) error
	ListPageSectionsByPageID(ctx context.Context, pageID int64) ([]PageSection, error)
	GetPageSectionByID(ctx context.Context, id int64) (*PageSection, error)
	UpdatePageSection(ctx context.Context, section *PageSection) error
	DeletePageSection(ctx context.Context, id int64) error
	DeletePageSectionsByPageID(ctx context.Context, pageID int64) error
}

type pgRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

const pageColumns = `id, title, slug, content, meta_description, featured_image, status, published_at, created_by, created_at, updated_at, deleted_at`
const blogPostColumns = `id, title, slug, content, excerpt, featured_image, author_id, status, published_at, tags, view_count, created_at, updated_at, deleted_at`
const pageSectionColumns = `id, page_id, type, title, content, sort_order, created_at, updated_at`

// --- Pages ---

func scanPage(sc interface{ Scan(dest ...interface{}) error }) (Page, error) {
	var p Page
	err := sc.Scan(
		&p.ID, &p.Title, &p.Slug, &p.Content, &p.MetaDescription, &p.FeaturedImage,
		&p.Status, &p.PublishedAt, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
	)
	return p, err
}

func (r *pgRepository) CreatePage(ctx context.Context, page *Page) error {
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx,
		`INSERT INTO cms_pages (title, slug, content, meta_description, featured_image, status, created_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, created_at, updated_at`,
		page.Title, page.Slug, page.Content, page.MetaDescription, page.FeaturedImage, page.Status, page.CreatedBy,
	).Scan(&page.ID, &page.CreatedAt, &page.UpdatedAt)
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

func (r *pgRepository) ListPages(ctx context.Context, limit int, cursor int64) ([]Page, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT `+pageColumns+` FROM cms_pages WHERE id > $2 AND deleted_at IS NULL ORDER BY id ASC LIMIT $1`,
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

func (r *pgRepository) UpdatePage(ctx context.Context, page *Page) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_pages SET title = $1, slug = $2, content = $3, meta_description = $4, featured_image = $5, updated_at = CURRENT_TIMESTAMP WHERE id = $6 AND deleted_at IS NULL`,
		page.Title, page.Slug, page.Content, page.MetaDescription, page.FeaturedImage, page.ID,
	)
	return err
}

func (r *pgRepository) DeletePage(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_pages SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) PublishPage(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_pages SET status = 'published', published_at = COALESCE(published_at, CURRENT_TIMESTAMP), updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) UnpublishPage(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_pages SET status = 'draft', updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	return err
}

// --- Blog Posts ---

func scanBlogPost(sc interface{ Scan(dest ...interface{}) error }) (BlogPost, error) {
	var bp BlogPost
	err := sc.Scan(
		&bp.ID, &bp.Title, &bp.Slug, &bp.Content, &bp.Excerpt, &bp.FeaturedImage,
		&bp.AuthorID, &bp.Status, &bp.PublishedAt, pq.Array(&bp.Tags),
		&bp.ViewCount, &bp.CreatedAt, &bp.UpdatedAt, &bp.DeletedAt,
	)
	if err != nil {
		return bp, err
	}
	if bp.Tags == nil {
		bp.Tags = []string{}
	}
	return bp, nil
}

func (r *pgRepository) CreateBlogPost(ctx context.Context, post *BlogPost) error {
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx,
		`INSERT INTO cms_blog_posts (title, slug, content, excerpt, featured_image, author_id, status, tags)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, created_at, updated_at`,
		post.Title, post.Slug, post.Content, post.Excerpt, post.FeaturedImage,
		post.AuthorID, post.Status, pq.Array(post.Tags),
	).Scan(&post.ID, &post.CreatedAt, &post.UpdatedAt)
}

func (r *pgRepository) GetBlogPostByID(ctx context.Context, id int64) (*BlogPost, error) {
	q := database.GetQueryer(ctx, r.db)
	bp, err := scanBlogPost(q.QueryRowContext(ctx,
		`SELECT `+blogPostColumns+` FROM cms_blog_posts WHERE id = $1 AND deleted_at IS NULL`, id,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &bp, nil
}

func (r *pgRepository) GetBlogPostBySlug(ctx context.Context, slug string) (*BlogPost, error) {
	q := database.GetQueryer(ctx, r.db)
	bp, err := scanBlogPost(q.QueryRowContext(ctx,
		`SELECT `+blogPostColumns+` FROM cms_blog_posts WHERE slug = $1 AND deleted_at IS NULL`, slug,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &bp, nil
}

func (r *pgRepository) ListBlogPosts(ctx context.Context, limit int, cursor int64) ([]BlogPost, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT `+blogPostColumns+` FROM cms_blog_posts WHERE id > $2 AND deleted_at IS NULL ORDER BY id ASC LIMIT $1`,
		limit, cursor,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []BlogPost
	for rows.Next() {
		bp, err := scanBlogPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, bp)
	}
	return posts, rows.Err()
}

func (r *pgRepository) ListPublishedBlogPosts(ctx context.Context, limit int, cursor int64) ([]BlogPost, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT `+blogPostColumns+` FROM cms_blog_posts WHERE id > $2 AND deleted_at IS NULL AND status = 'published' ORDER BY id ASC LIMIT $1`,
		limit, cursor,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []BlogPost
	for rows.Next() {
		bp, err := scanBlogPost(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, bp)
	}
	return posts, rows.Err()
}

func (r *pgRepository) UpdateBlogPost(ctx context.Context, post *BlogPost) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_blog_posts SET title = $1, slug = $2, content = $3, excerpt = $4, featured_image = $5, tags = $6, updated_at = CURRENT_TIMESTAMP WHERE id = $7 AND deleted_at IS NULL`,
		post.Title, post.Slug, post.Content, post.Excerpt, post.FeaturedImage, pq.Array(post.Tags), post.ID,
	)
	return err
}

func (r *pgRepository) DeleteBlogPost(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_blog_posts SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) PublishBlogPost(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_blog_posts SET status = 'published', published_at = COALESCE(published_at, CURRENT_TIMESTAMP), updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) UnpublishBlogPost(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_blog_posts SET status = 'draft', updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	return err
}

func (r *pgRepository) IncrementViewCount(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_blog_posts SET view_count = view_count + 1 WHERE id = $1 AND deleted_at IS NULL`, id,
	)
	return err
}

// --- Page Sections ---

func scanPageSection(sc interface{ Scan(dest ...interface{}) error }) (PageSection, error) {
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

func (r *pgRepository) CreatePageSection(ctx context.Context, section *PageSection) error {
	q := database.GetQueryer(ctx, r.db)
	return q.QueryRowContext(ctx,
		`INSERT INTO cms_page_sections (page_id, type, title, content, sort_order)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		section.PageID, section.Type, section.Title, []byte(section.Content), section.SortOrder,
	).Scan(&section.ID, &section.CreatedAt, &section.UpdatedAt)
}

func (r *pgRepository) ListPageSectionsByPageID(ctx context.Context, pageID int64) ([]PageSection, error) {
	q := database.GetQueryer(ctx, r.db)
	rows, err := q.QueryContext(ctx,
		`SELECT `+pageSectionColumns+` FROM cms_page_sections WHERE page_id = $1 ORDER BY sort_order ASC`, pageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sections []PageSection
	for rows.Next() {
		ps, err := scanPageSection(rows)
		if err != nil {
			return nil, err
		}
		sections = append(sections, ps)
	}
	return sections, rows.Err()
}

func (r *pgRepository) GetPageSectionByID(ctx context.Context, id int64) (*PageSection, error) {
	q := database.GetQueryer(ctx, r.db)
	ps, err := scanPageSection(q.QueryRowContext(ctx,
		`SELECT `+pageSectionColumns+` FROM cms_page_sections WHERE id = $1`, id,
	))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &ps, nil
}

func (r *pgRepository) UpdatePageSection(ctx context.Context, section *PageSection) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx,
		`UPDATE cms_page_sections SET type = $1, title = $2, content = $3, sort_order = $4, updated_at = CURRENT_TIMESTAMP WHERE id = $5`,
		section.Type, section.Title, []byte(section.Content), section.SortOrder, section.ID,
	)
	return err
}

func (r *pgRepository) DeletePageSection(ctx context.Context, id int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, `DELETE FROM cms_page_sections WHERE id = $1`, id)
	return err
}

func (r *pgRepository) DeletePageSectionsByPageID(ctx context.Context, pageID int64) error {
	q := database.GetQueryer(ctx, r.db)
	_, err := q.ExecContext(ctx, `DELETE FROM cms_page_sections WHERE page_id = $1`, pageID)
	return err
}
