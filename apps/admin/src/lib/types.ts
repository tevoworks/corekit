export interface User {
  id: number
  email: string
  full_name: string
  is_super_admin: boolean
  role_id: number | null
  role_name: string | null
  avatar_url: string
  status: string
  created_at: string
  updated_at: string
  deleted_at: string | null
}

export interface Session {
  id: number
  user_id: number
  token_id: string
  ip_address: string
  user_agent: string
  created_at: string
  expires_at: string
  revoked_at: string | null
}

export interface Role {
  id: number
  name: string
  description: string
  created_at: string
  updated_at: string
}

export interface Permission {
  id: number
  name: string
  description: string
  created_at: string
}

export interface Setting {
  id: number
  key: string
  value: string
  created_at: string
  updated_at: string
}

export interface FeatureFlag {
  id: number
  name: string
  key: string
  enabled: boolean
  description: string
  created_at: string
  updated_at: string
}

export interface AuditLog {
  id: number
  actor_id: number | null
  action: string
  target_entity: string
  before_state: any
  after_state: any
  created_at: string
  actor_name: string | null
  actor_email: string | null
}

export interface APIKey {
  id: number
  name: string
  key_prefix: string
  created_by: number
  revoked_at: string | null
  last_used_at: string | null
  created_at: string
}

export interface Webhook {
  id: number
  name: string
  url: string
  events: string[]
  active: boolean
  created_by: number
  created_at: string
  updated_at: string
}

export interface WebhookDelivery {
  id: number
  webhook_id: number
  event: string
  status: string
  request_body: string | null
  response_body: string | null
  response_code: number | null
  duration_ms: number | null
  error_message: string | null
  created_at: string
}

export interface PermissionRegistry {
  id: number
  domain: string
  name: string
  description: string
  created_at: string
  updated_at: string
}

export interface GlobalTemplate {
  id: number
  name: string
  description: string
  permissions: string[]
  created_at: string
  updated_at: string
}

export interface Job {
  id: number
  type: string
  payload: any
  status: string
  priority: number
  retry_count: number
  max_retries: number
  run_after: string
  error_message: string | null
  created_at: string
  updated_at: string
}

export interface FileItem {
  id: number
  filename: string
  mime_type: string
  size_bytes: number
  storage_path: string
  url: string
  checksum_sha256: string
  uploaded_by: number | null
  is_public: boolean
  created_at: string
}

export interface Notification {
  id: number
  user_id: number
  type: string
  title: string
  body: string
  data: any
  is_read: boolean
  created_at: string
}

export interface UserPreference {
  user_id: number
  key: string
  value: string
  updated_at: string
}

export interface ApiResponse<T> {
  data: T
  meta?: {
    total?: number
    limit?: number
    offset?: number
    cursor?: string
  }
  error?: {
    code: string
    message: string
  }
}

export interface Page {
  id: number
  title: string
  slug: string
  content: string
  meta_title: string
  meta_description: string
  og_image: string
  featured_image_id: number | null
  featured_image: string
  status: string
  published_at: string | null
  created_by: number
  created_at: string
  updated_at: string
  deleted_at: string | null
}

export interface BlogPost {
  id: number
  title: string
  slug: string
  content: string
  excerpt: string
  meta_title: string
  meta_description: string
  og_image: string
  featured_image_id: number | null
  featured_image: string
  author_id: number
  status: string
  published_at: string | null
  tags: string[]
  view_count: number
  created_at: string
  updated_at: string
  deleted_at: string | null
}

export interface PageSection {
  id: number
  page_id: number
  type: string
  title: string
  content: any
  sort_order: number
  created_at: string
  updated_at: string
}

export interface Contact {
  id: number
  name: string
  email: string
  phone: string
  subject: string
  message: string
  source: string
  status: string
  assigned_to: number | null
  created_at: string
  updated_at: string
}

export interface NewsletterSubscriber {
  id: number
  email: string
  name: string
  source: string
  metadata: any
  subscribed_at: string
  unsubscribed_at: string | null
}
