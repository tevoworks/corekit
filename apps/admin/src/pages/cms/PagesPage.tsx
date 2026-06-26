import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { Page } from '../../lib/types'
import { useState } from 'react'
import { Card, PageHeader, Table, Button, Modal, Input, StatusBadge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'
import { useNavigate } from 'react-router-dom'

export default function PagesPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<Page | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState('')
  const [cursor, setCursor] = useState(0)
  const [allPages, setAllPages] = useState<Page[]>([])
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  const { isLoading } = useQuery({
    queryKey: ['CMS', 'pages', statusFilter, cursor],
    queryFn: () => api.get(`/api/cms/pages?status=${statusFilter}&limit=50&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data : []
      if (cursor === 0) {
        setAllPages(items)
      } else {
        setAllPages(prev => [...prev, ...items])
      }
      return { items, next_cursor: r.data.meta?.next_cursor || 0 }
    }),
  })

  const createMutation = useMutation({
    mutationFn: (body: any) => api.post('/api/cms/pages', body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'pages'] }); setShowForm(false) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to create page'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, ...body }: any) => api.put(`/api/cms/pages/${id}`, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'pages'] }); setEditing(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to update page'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/cms/pages/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'pages'] }); setCursor(0); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete page'),
  })

  const publishMutation = useMutation({
    mutationFn: ({ id, publish }: { id: number; publish: boolean }) =>
      api.post(`/api/cms/pages/${id}/${publish ? 'publish' : 'unpublish'}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'pages'] }) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to change page status'),
  })

  const handleFilterChange = (val: string) => {
    setStatusFilter(val)
    setCursor(0)
  }

  if (isLoading && allPages.length === 0) return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="CMS Pages" description="Manage website pages" />
      <Card testId="pages-loading-card"><LoadingSkeleton rows={5} testId="pages-loading" /></Card>
    </main>
  )

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="CMS Pages"
        description="Manage website pages"
        action={
          <Button icon="add" onClick={() => setShowForm(true)} data-testid="pages-add-button">Add Page</Button>
        }
      />

      <div className="mb-4 flex items-center gap-3">
        <select
          value={statusFilter}
          onChange={e => handleFilterChange(e.target.value)}
          className="rounded-lg border border-[var(--outline-variant)] px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
          data-testid="pages-status-filter"
        >
          <option value="">All Statuses</option>
          <option value="draft">Draft</option>
          <option value="published">Published</option>
        </select>
      </div>

      {showForm && (
        <Modal title="Create Page" onClose={() => { setShowForm(false); setError(null) }} testId="pages-create-modal">
          <PageFormContent
            onSave={(body) => createMutation.mutate(body)}
            onCancel={() => { setShowForm(false); setError(null) }}
            saving={createMutation.isPending}
            error={error}
            onClearError={() => setError(null)}
          />
        </Modal>
      )}
      {editing && (
        <Modal title="Edit Page" onClose={() => { setEditing(null); setError(null) }} testId="pages-edit-modal">
          <PageFormContent
            page={editing}
            onSave={(body) => updateMutation.mutate({ id: editing.id, ...body })}
            onCancel={() => { setEditing(null); setError(null) }}
            saving={updateMutation.isPending}
            error={error}
            onClearError={() => setError(null)}
          />
        </Modal>
      )}

      {allPages.length === 0 ? (
        <Card testId="pages-empty-card"><EmptyState icon="description" title="No pages yet" description="Create your first page to get started." action={<Button icon="add" variant="secondary" size="sm" onClick={() => setShowForm(true)} data-testid="pages-empty-add-button">Add Page</Button>} testId="pages-empty-state" /></Card>
      ) : (
        <>
          <Table headers={['Title', 'Slug', 'Status', 'Updated At', 'Actions']} testId="pages-table">
            {allPages.map((p: Page) => (
              <tr key={p.id} className="hover:bg-[var(--surface-container-low)] transition-colors cursor-pointer" data-testid={`pages-row-${p.id}`} onClick={() => navigate(`/cms/pages/${p.id}/sections`)}>
                <td className="p-3 font-medium">{p.title}</td>
                <td className="p-3 text-[var(--on-surface-variant)]">/{p.slug}</td>
                <td className="p-3"><StatusBadge status={p.status} /></td>
                <td className="p-3 text-[var(--on-surface-variant)] text-xs">{new Date(p.updated_at).toLocaleDateString()}</td>
                <td className="p-3 text-right" onClick={e => e.stopPropagation()}>
                  <Button variant="secondary" size="sm" icon="edit" onClick={() => setEditing(p)} data-testid={`pages-edit-${p.id}`} className="mr-1">Edit</Button>
                  <Button variant="secondary" size="sm" icon={p.status === 'published' ? 'unpublished' : 'publish'} onClick={() => publishMutation.mutate({ id: p.id, publish: p.status !== 'published' })} data-testid={`pages-publish-${p.id}`} className="mr-1">{p.status === 'published' ? 'Unpublish' : 'Publish'}</Button>
                  <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(p.id)} data-testid={`pages-delete-${p.id}`}>Delete</Button>
                </td>
              </tr>
            ))}
          </Table>
          {allPages.length >= 50 && (
            <div className="mt-4 text-center">
              <Button variant="secondary" size="sm" onClick={() => setCursor(c => c + 50)} data-testid="pages-load-more-button">Load More</Button>
            </div>
          )}
        </>
      )}

      {confirmDelete && (
        <ConfirmDialog
          open={!!confirmDelete}
          onClose={() => { setConfirmDelete(null); setError(null) }}
          onConfirm={() => deleteMutation.mutate(confirmDelete)}
          title="Delete Page"
          message="This will permanently remove this page and all its sections."
          confirmLabel="Yes, Permanently Delete"
          loading={deleteMutation.isPending}
          error={error}
          testId="pages-delete-confirm-dialog"
        />
      )}
    </main>
  )
}

function PageFormContent({ page, onSave, onCancel, saving, error, onClearError }: {
  page?: Page | null
  onSave: (body: any) => void
  onCancel: () => void
  saving?: boolean
  error?: string | null
  onClearError?: () => void
}) {
  const [title, setTitle] = useState(page?.title || '')
  const [slug, setSlug] = useState(page?.slug || '')
  const [content, setContent] = useState(page?.content || '')
  const [metaDescription, setMetaDescription] = useState(page?.meta_description || '')
  const [featuredImage, setFeaturedImage] = useState(page?.featured_image || '')

  return (
    <div>
      {error && (
        <div className="mb-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm flex items-center gap-2" data-testid="pages-form-error">
          <span className="material-symbols-outlined text-base">error</span>
          <span className="flex-1">{error}</span>
          <button onClick={onClearError} className="text-[var(--danger-text)] hover:opacity-70">
            <span className="material-symbols-outlined text-base">close</span>
          </button>
        </div>
      )}
      <div className="space-y-4">
        <Input label="Title *" value={title} onChange={e => setTitle(e.target.value)} required data-testid="pages-form-title" />
        <Input label="Slug" value={slug} onChange={e => setSlug(e.target.value)} data-testid="pages-form-slug" placeholder="leave empty to auto-generate" />
        <div>
          <label className="block text-sm font-medium text-[var(--on-surface)] mb-1">Content</label>
          <textarea value={content} onChange={e => setContent(e.target.value)} rows={6} className="w-full px-3 py-3 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent transition-all" data-testid="pages-form-content" />
        </div>
        <Input label="Meta Description" value={metaDescription} onChange={e => setMetaDescription(e.target.value)} data-testid="pages-form-meta" />
        <Input label="Featured Image URL" value={featuredImage} onChange={e => setFeaturedImage(e.target.value)} data-testid="pages-form-image" />
      </div>
      <div className="flex gap-2 justify-end mt-6 pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onCancel} data-testid="pages-form-cancel">Cancel</Button>
        <Button onClick={() => onSave({ title, slug, content, meta_description: metaDescription, featured_image: featuredImage })} loading={saving} disabled={!title} data-testid="pages-form-submit">{page ? 'Update' : 'Create'}</Button>
      </div>
    </div>
  )
}
