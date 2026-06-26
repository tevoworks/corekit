export interface Page {
  id: number
  title: string
  slug: string
  content: string
  meta_description: string
  featured_image: string
  status: string
  published_at: string | null
  created_by: number
  created_at: string
  updated_at: string
}

export interface BlogPost {
  id: number
  title: string
  slug: string
  content: string
  excerpt: string
  featured_image: string
  author_id: number
  status: string
  published_at: string | null
  tags: string[]
  view_count: number
  created_at: string
  updated_at: string
}

export interface PageSection {
  id: number
  page_id: number
  type: string
  title: string
  content: Record<string, unknown>
  sort_order: number
  created_at: string
  updated_at: string
}

export interface ApiResponse<T> {
  data: T
  meta?: { limit?: number; cursor?: number; count?: number }
  error?: { code: string; message: string }
}
