package cms

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/microcosm-cc/bluemonday"
	"github.com/tevoworks/corekit/backend/internal/database"
	"github.com/tevoworks/corekit/backend/internal/modules/audit"
)

var strictPolicy = bluemonday.StrictPolicy()
var ErrSlugConflict = errors.New("slug already exists")

type Service interface {
	CreatePage(ctx context.Context, title, slug, content, metaTitle, metaDescription, ogImage string, featuredImageID *int64, actorID int64) (*Page, error)
	UpdatePage(ctx context.Context, id int64, title, slug, content, metaTitle, metaDescription, ogImage string, featuredImageID *int64, actorID int64) (*Page, error)
	GetPage(ctx context.Context, id int64) (*Page, error)
	GetPageBySlug(ctx context.Context, slug string) (*Page, error)
	ListPages(ctx context.Context, limit, cursor int64, status string) ([]Page, error)
	DeletePage(ctx context.Context, id, actorID int64) error
	PublishPage(ctx context.Context, id, actorID int64) error
	UnpublishPage(ctx context.Context, id, actorID int64) error
	ListPublishedPages(ctx context.Context, limit, cursor int64) ([]Page, error)
	CheckSlugExists(ctx context.Context, slug string, excludeID int64) (bool, error)

	CreatePost(ctx context.Context, title, slug, content, excerpt, metaTitle, metaDescription, ogImage string, featuredImageID *int64, tags []string, authorID int64) (*BlogPost, error)
	UpdatePost(ctx context.Context, id int64, title, slug, content, excerpt, metaTitle, metaDescription, ogImage string, featuredImageID *int64, tags []string, actorID int64) (*BlogPost, error)
	GetPost(ctx context.Context, id int64) (*BlogPost, error)
	GetPostBySlug(ctx context.Context, slug string) (*BlogPost, error)
	ListPosts(ctx context.Context, limit, cursor int64, status string) ([]BlogPost, error)
	DeletePost(ctx context.Context, id, actorID int64) error
	PublishPost(ctx context.Context, id, actorID int64) error
	UnpublishPost(ctx context.Context, id, actorID int64) error
	ListPublishedPosts(ctx context.Context, limit, cursor int64) ([]BlogPost, error)
	IncrementViewCount(ctx context.Context, id int64) error

	CreateSection(ctx context.Context, pageID int64, sectionType, title string, content json.RawMessage, sortOrder int, actorID int64) (*PageSection, error)
	UpdateSection(ctx context.Context, id, pageID int64, sectionType, title string, content json.RawMessage, sortOrder int, actorID int64) (*PageSection, error)
	GetSection(ctx context.Context, id int64) (*PageSection, error)
	ListSectionsByPage(ctx context.Context, pageID int64) ([]PageSection, error)
	DeleteSection(ctx context.Context, id, actorID int64) error
}

type service struct {
	db           *sql.DB
	repo         Repository
	auditService audit.Service
}

func NewService(db *sql.DB, repo Repository, auditService audit.Service) Service {
	return &service{db: db, repo: repo, auditService: auditService}
}

// --- Pages ---

func (s *service) CreatePage(ctx context.Context, title, slug, content, metaTitle, metaDescription, ogImage string, featuredImageID *int64, actorID int64) (*Page, error) {
	exists, err := s.repo.CheckSlugExists(ctx, slug, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSlugConflict
	}

	p := &Page{
		Title:           title,
		Slug:            slug,
		Content:         strictPolicy.Sanitize(content),
		MetaTitle:       metaTitle,
		MetaDescription: metaDescription,
		OgImage:         ogImage,
		FeaturedImageID: featuredImageID,
		Status:          "draft",
		CreatedBy:       actorID,
	}

	actx := database.WithAuditCtx(ctx, actorID, "CREATE_PAGE")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.CreatePage(txCtx, p)
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) UpdatePage(ctx context.Context, id int64, title, slug, content, metaTitle, metaDescription, ogImage string, featuredImageID *int64, actorID int64) (*Page, error) {
	var p *Page
	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_PAGE")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		var err error
		p, err = s.repo.GetPageByID(txCtx, id)
		if err != nil {
			return err
		}
		if p == nil {
			return database.ErrNotFound
		}

		if slug != p.Slug {
			exists, err := s.repo.CheckSlugExists(txCtx, slug, id)
			if err != nil {
				return err
			}
			if exists {
				return ErrSlugConflict
			}
		}

		p.Title = title
		p.Slug = slug
		p.Content = strictPolicy.Sanitize(content)
		p.MetaTitle = metaTitle
		p.MetaDescription = metaDescription
		p.OgImage = ogImage
		p.FeaturedImageID = featuredImageID
		return s.repo.UpdatePage(txCtx, p)
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) GetPage(ctx context.Context, id int64) (*Page, error) {
	return s.repo.GetPageByID(ctx, id)
}

func (s *service) GetPageBySlug(ctx context.Context, slug string) (*Page, error) {
	return s.repo.GetPageBySlug(ctx, slug)
}

func (s *service) ListPages(ctx context.Context, limit, cursor int64, status string) ([]Page, error) {
	return s.repo.ListPages(ctx, int(limit), cursor, status)
}

func (s *service) DeletePage(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_PAGE")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.DeletePage(txCtx, id)
	})
}

func (s *service) PublishPage(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "PUBLISH_PAGE")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.PublishPage(txCtx, id)
	})
}

func (s *service) UnpublishPage(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "UNPUBLISH_PAGE")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.UnpublishPage(txCtx, id)
	})
}

func (s *service) ListPublishedPages(ctx context.Context, limit, cursor int64) ([]Page, error) {
	return s.repo.ListPublishedPages(ctx, int(limit), cursor)
}

func (s *service) CheckSlugExists(ctx context.Context, slug string, excludeID int64) (bool, error) {
	return s.repo.CheckSlugExists(ctx, slug, excludeID)
}

// --- Blog Posts ---

func (s *service) CreatePost(ctx context.Context, title, slug, content, excerpt, metaTitle, metaDescription, ogImage string, featuredImageID *int64, tags []string, authorID int64) (*BlogPost, error) {
	exists, err := s.repo.CheckSlugExists(ctx, slug, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrSlugConflict
	}

	if tags == nil {
		tags = []string{}
	}

	p := &BlogPost{
		Title:           title,
		Slug:            slug,
		Content:         strictPolicy.Sanitize(content),
		Excerpt:         excerpt,
		MetaTitle:       metaTitle,
		MetaDescription: metaDescription,
		OgImage:         ogImage,
		FeaturedImageID: featuredImageID,
		AuthorID:        authorID,
		Status:          "draft",
		Tags:            tags,
	}

	actx := database.WithAuditCtx(ctx, authorID, "CREATE_POST")
	err = database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.CreatePost(txCtx, p)
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) UpdatePost(ctx context.Context, id int64, title, slug, content, excerpt, metaTitle, metaDescription, ogImage string, featuredImageID *int64, tags []string, actorID int64) (*BlogPost, error) {
	var p *BlogPost
	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_POST")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		var err error
		p, err = s.repo.GetPostByID(txCtx, id)
		if err != nil {
			return err
		}
		if p == nil {
			return database.ErrNotFound
		}

		if slug != p.Slug {
			exists, err := s.repo.CheckSlugExists(txCtx, slug, id)
			if err != nil {
				return err
			}
			if exists {
				return ErrSlugConflict
			}
		}

		p.Title = title
		p.Slug = slug
		p.Content = strictPolicy.Sanitize(content)
		p.Excerpt = excerpt
		p.MetaTitle = metaTitle
		p.MetaDescription = metaDescription
		p.OgImage = ogImage
		p.FeaturedImageID = featuredImageID
		p.Tags = tags
		if p.Tags == nil {
			p.Tags = []string{}
		}
		return s.repo.UpdatePost(txCtx, p)
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *service) GetPost(ctx context.Context, id int64) (*BlogPost, error) {
	return s.repo.GetPostByID(ctx, id)
}

func (s *service) GetPostBySlug(ctx context.Context, slug string) (*BlogPost, error) {
	return s.repo.GetPostBySlug(ctx, slug)
}

func (s *service) ListPosts(ctx context.Context, limit, cursor int64, status string) ([]BlogPost, error) {
	return s.repo.ListPosts(ctx, int(limit), cursor, status)
}

func (s *service) DeletePost(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_POST")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.DeletePost(txCtx, id)
	})
}

func (s *service) PublishPost(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "PUBLISH_POST")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.PublishPost(txCtx, id)
	})
}

func (s *service) UnpublishPost(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "UNPUBLISH_POST")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.UnpublishPost(txCtx, id)
	})
}

func (s *service) ListPublishedPosts(ctx context.Context, limit, cursor int64) ([]BlogPost, error) {
	return s.repo.ListPublishedPosts(ctx, int(limit), cursor)
}

func (s *service) IncrementViewCount(ctx context.Context, id int64) error {
	return s.repo.IncrementPostViewCount(ctx, id)
}

// --- Page Sections ---

func (s *service) CreateSection(ctx context.Context, pageID int64, sectionType, title string, content json.RawMessage, sortOrder int, actorID int64) (*PageSection, error) {
	sec := &PageSection{
		PageID:    pageID,
		Type:      sectionType,
		Title:     title,
		Content:   content,
		SortOrder: sortOrder,
	}

	actx := database.WithAuditCtx(ctx, actorID, "CREATE_SECTION")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.CreateSection(txCtx, sec)
	})
	if err != nil {
		return nil, err
	}
	return sec, nil
}

func (s *service) UpdateSection(ctx context.Context, id, pageID int64, sectionType, title string, content json.RawMessage, sortOrder int, actorID int64) (*PageSection, error) {
	var sec *PageSection
	actx := database.WithAuditCtx(ctx, actorID, "UPDATE_SECTION")
	err := database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		var err error
		sec, err = s.repo.GetSectionByID(txCtx, id)
		if err != nil {
			return err
		}
		if sec == nil {
			return database.ErrNotFound
		}
		sec.PageID = pageID
		sec.Type = sectionType
		sec.Title = title
		sec.Content = content
		sec.SortOrder = sortOrder
		return s.repo.UpdateSection(txCtx, sec)
	})
	if err != nil {
		return nil, err
	}
	return sec, nil
}

func (s *service) GetSection(ctx context.Context, id int64) (*PageSection, error) {
	return s.repo.GetSectionByID(ctx, id)
}

func (s *service) ListSectionsByPage(ctx context.Context, pageID int64) ([]PageSection, error) {
	return s.repo.ListSectionsByPageID(ctx, pageID)
}

func (s *service) DeleteSection(ctx context.Context, id, actorID int64) error {
	actx := database.WithAuditCtx(ctx, actorID, "DELETE_SECTION")
	return database.RunInTransaction(actx, s.db, func(txCtx context.Context) error {
		return s.repo.DeleteSection(txCtx, id)
	})
}
