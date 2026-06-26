-- 000013: Create CMS tables (pages, blog_posts, page_sections)

CREATE TABLE IF NOT EXISTS pages (
    id                BIGSERIAL PRIMARY KEY,
    title             VARCHAR(255) NOT NULL,
    slug              VARCHAR(255) NOT NULL UNIQUE,
    content           TEXT NOT NULL DEFAULT '',
    meta_title        VARCHAR(255) NOT NULL DEFAULT '',
    meta_description  TEXT,
    og_image          VARCHAR(500) NOT NULL DEFAULT '',
    featured_image_id BIGINT REFERENCES file_metadata(id),
    status            VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    published_at      TIMESTAMP,
    created_by        BIGINT NOT NULL REFERENCES users(id),
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        TIMESTAMP
);

CREATE TABLE IF NOT EXISTS blog_posts (
    id                BIGSERIAL PRIMARY KEY,
    title             VARCHAR(255) NOT NULL,
    slug              VARCHAR(255) NOT NULL UNIQUE,
    content           TEXT NOT NULL DEFAULT '',
    excerpt           TEXT,
    meta_title        VARCHAR(255) NOT NULL DEFAULT '',
    meta_description  TEXT,
    og_image          VARCHAR(500) NOT NULL DEFAULT '',
    featured_image_id BIGINT REFERENCES file_metadata(id),
    author_id         BIGINT NOT NULL REFERENCES users(id),
    status            VARCHAR(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    published_at      TIMESTAMP,
    tags              TEXT[] DEFAULT '{}',
    view_count        INT NOT NULL DEFAULT 0,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        TIMESTAMP
);

CREATE TABLE IF NOT EXISTS page_sections (
    id         BIGSERIAL PRIMARY KEY,
    page_id    BIGINT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    type       VARCHAR(50) NOT NULL,
    title      VARCHAR(255) NOT NULL DEFAULT '',
    content    JSONB NOT NULL DEFAULT '{}',
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pages_slug ON pages(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_pages_status ON pages(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_blog_posts_slug ON blog_posts(slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_blog_posts_status ON blog_posts(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_blog_posts_published_at ON blog_posts(published_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_page_sections_page_id ON page_sections(page_id);
