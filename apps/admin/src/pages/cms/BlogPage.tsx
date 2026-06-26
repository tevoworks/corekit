import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { BlogPost } from '../../lib/types'
import { useState } from 'react'
import { Card, PageHeader, Table, Button, Modal, Input, StatusBadge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

export default function BlogPage() {
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<BlogPost | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState('')
  const [cursor, setCursor] = useState(0)
  const [allPosts, setAllPosts] = useState<BlogPost[]>([])
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  const { isLoading } = useQuery({
    queryKey: ['CMS', 'blog', statusFilter, cursor],
    queryFn: () => api.get(`/api/cms/blog?status=${statusFilter}&limit=50&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data : []
      if (cursor === 0) {
        setAllPosts(items)
      } else {
        setAllPosts(prev => [...prev, ...items])
      }
      return { items, next_cursor: r.data.meta?.next_cursor || 0 }
    }),
  })

  const createMutation = useMutation({
    mutationFn: (body: any) => api.post('/api/cms/blog', body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'blog'] }); setShowForm(false) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to create blog post'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, ...body }: any) => api.put(`/api/cms/blog/${id}`, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'blog'] }); setEditing(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to update blog post'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/cms/blog/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'blog'] }); setCursor(0); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete blog post'),
  })

  const publishMutation = useMutation({
    mutationFn: ({ id, publish }: { id: number; publish: boolean }) =>
      api.post(`/api/cms/blog/${id}/${publish ? 'publish' : 'unpublish'}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'blog'] }) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to change post status'),
  })

  const handleFilterChange = (val: string) => {
    setStatusFilter(val)
    setCursor(0)
  }

  if (isLoading && allPosts.length === 0) return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="Blog Posts" description="Manage blog content" />
      <Card testId="blog-loading-card"><LoadingSkeleton rows={5} testId="blog-loading" /></Card>
    </main>
  )

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="Blog Posts"
        description="Manage blog content"
        action={
          <Button icon="add" onClick={() => setShowForm(true)} data-testid="blog-add-button">Add Post</Button>
        }
      />

      <div className="mb-4 flex items-center gap-3">
        <select
          value={statusFilter}
          onChange={e => handleFilterChange(e.target.value)}
          className="rounded-lg border border-[var(--outline-variant)] px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
          data-testid="blog-status-filter"
        >
          <option value="">All Statuses</option>
          <option value="draft">Draft</option>
          <option value="published">Published</option>
        </select>
      </div>

      {showForm && (
        <Modal title="Create Blog Post" onClose={() => { setShowForm(false); setError(null) }} testId="blog-create-modal">
          <BlogFormContent
            onSave={(body) => createMutation.mutate(body)}
            onCancel={() => { setShowForm(false); setError(null) }}
            saving={createMutation.isPending}
            error={error}
            onClearError={() => setError(null)}
          />
        </Modal>
      )}
      {editing && (
        <Modal title="Edit Blog Post" onClose={() => { setEditing(null); setError(null) }} testId="blog-edit-modal">
          <BlogFormContent
            post={editing}
            onSave={(body) => updateMutation.mutate({ id: editing.id, ...body })}
            onCancel={() => { setEditing(null); setError(null) }}
            saving={updateMutation.isPending}
            error={error}
            onClearError={() => setError(null)}
          />
        </Modal>
      )}

      {allPosts.length === 0 ? (
        <Card testId="blog-empty-card"><EmptyState icon="article" title="No blog posts yet" description="Create your first post to get started." action={<Button icon="add" variant="secondary" size="sm" onClick={() => setShowForm(true)} data-testid="blog-empty-add-button">Add Post</Button>} testId="blog-empty-state" /></Card>
      ) : (
        <>
          <Table headers={['Title', 'Slug', 'Status', 'Tags', 'Published At', 'Actions']} testId="blog-table">
            {allPosts.map((p: BlogPost) => (
              <tr key={p.id} className="hover:bg-[var(--surface-container-low)] transition-colors" data-testid={`blog-row-${p.id}`}>
                <td className="p-3 font-medium">{p.title}</td>
                <td className="p-3 text-[var(--on-surface-variant)]">/{p.slug}</td>
                <td className="p-3"><StatusBadge status={p.status} /></td>
                <td className="p-3 text-[var(--on-surface-variant)] text-xs">{(p.tags || []).join(', ') || '—'}</td>
                <td className="p-3 text-[var(--on-surface-variant)] text-xs">{p.published_at ? new Date(p.published_at).toLocaleDateString() : '—'}</td>
                <td className="p-3 text-right">
                  <Button variant="secondary" size="sm" icon="edit" onClick={() => setEditing(p)} data-testid={`blog-edit-${p.id}`} className="mr-1">Edit</Button>
                  <Button variant="secondary" size="sm" icon={p.status === 'published' ? 'unpublished' : 'publish'} onClick={() => publishMutation.mutate({ id: p.id, publish: p.status !== 'published' })} data-testid={`blog-publish-${p.id}`} className="mr-1">{p.status === 'published' ? 'Unpublish' : 'Publish'}</Button>
                  <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(p.id)} data-testid={`blog-delete-${p.id}`}>Delete</Button>
                </td>
              </tr>
            ))}
          </Table>
          {allPosts.length >= 50 && (
            <div className="mt-4 text-center">
              <Button variant="secondary" size="sm" onClick={() => setCursor(c => c + 50)} data-testid="blog-load-more-button">Load More</Button>
            </div>
          )}
        </>
      )}

      {confirmDelete && (
        <ConfirmDialog
          open={!!confirmDelete}
          onClose={() => { setConfirmDelete(null); setError(null) }}
          onConfirm={() => deleteMutation.mutate(confirmDelete)}
          title="Delete Blog Post"
          message="This will permanently remove this blog post."
          confirmLabel="Yes, Permanently Delete"
          loading={deleteMutation.isPending}
          error={error}
          testId="blog-delete-confirm-dialog"
        />
      )}
    </main>
  )
}

function BlogFormContent({ post, onSave, onCancel, saving, error, onClearError }: {
  post?: BlogPost | null
  onSave: (body: any) => void
  onCancel: () => void
  saving?: boolean
  error?: string | null
  onClearError?: () => void
}) {
  const [title, setTitle] = useState(post?.title || '')
  const [slug, setSlug] = useState(post?.slug || '')
  const [content, setContent] = useState(post?.content || '')
  const [excerpt, setExcerpt] = useState(post?.excerpt || '')
  const [featuredImage, setFeaturedImage] = useState(post?.featured_image || '')
  const [tags, setTags] = useState((post?.tags || []).join(', '))

  return (
    <div>
      {error && (
        <div className="mb-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm flex items-center gap-2" data-testid="blog-form-error">
          <span className="material-symbols-outlined text-base">error</span>
          <span className="flex-1">{error}</span>
          <button onClick={onClearError} className="text-[var(--danger-text)] hover:opacity-70">
            <span className="material-symbols-outlined text-base">close</span>
          </button>
        </div>
      )}
      <div className="space-y-4">
        <Input label="Title *" value={title} onChange={e => setTitle(e.target.value)} required data-testid="blog-form-title" />
        <Input label="Slug" value={slug} onChange={e => setSlug(e.target.value)} data-testid="blog-form-slug" placeholder="leave empty to auto-generate" />
        <div>
          <label className="block text-sm font-medium text-[var(--on-surface)] mb-1">Content</label>
          <textarea value={content} onChange={e => setContent(e.target.value)} rows={6} className="w-full px-3 py-3 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent transition-all" data-testid="blog-form-content" />
        </div>
        <div>
          <label className="block text-sm font-medium text-[var(--on-surface)] mb-1">Excerpt</label>
          <textarea value={excerpt} onChange={e => setExcerpt(e.target.value)} rows={3} className="w-full px-3 py-3 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent transition-all" data-testid="blog-form-excerpt" />
        </div>
        <Input label="Featured Image URL" value={featuredImage} onChange={e => setFeaturedImage(e.target.value)} data-testid="blog-form-image" />
        <Input label="Tags (comma separated)" value={tags} onChange={e => setTags(e.target.value)} data-testid="blog-form-tags" />
      </div>
      <div className="flex gap-2 justify-end mt-6 pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onCancel} data-testid="blog-form-cancel">Cancel</Button>
        <Button onClick={() => onSave({ title, slug, content, excerpt, featured_image: featuredImage, tags: tags.split(',').map(t => t.trim()).filter(Boolean) })} loading={saving} disabled={!title} data-testid="blog-form-submit">{post ? 'Update' : 'Create'}</Button>
      </div>
    </div>
  )
}
