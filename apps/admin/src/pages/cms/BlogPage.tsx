import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api, { getApiError } from '../../lib/api'
import type { BlogPost } from '../../lib/types'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, PageHeader, Table, Button, StatusBadge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

export default function BlogPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [statusFilter, setStatusFilter] = useState('')
  const [cursor, setCursor] = useState(0)
  const [allPosts, setAllPosts] = useState<BlogPost[]>([])
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)
  const [deleteErr, setDeleteErr] = useState<string | null>(null)

  const { isLoading } = useQuery({
    queryKey: ['CMS', 'blog', statusFilter, cursor],
    queryFn: () => api.get(`/api/cms/posts?status=${statusFilter}&limit=50&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data : []
      if (cursor === 0) setAllPosts(items)
      else setAllPosts(prev => [...prev, ...items])
      return { items, next_cursor: r.data.meta?.next_cursor || 0 }
    }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/cms/posts/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'blog'] }); setCursor(0); setConfirmDelete(null) },
    onError: (err: any) => { setDeleteErr(getApiError(err)) },
  })

  const publishMutation = useMutation({
    mutationFn: ({ id, publish }: { id: number; publish: boolean }) => api.post(`/api/cms/posts/${id}/${publish ? 'publish' : 'unpublish'}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'blog'] }) },
    onError: (err: any) => { setDeleteErr(getApiError(err)) },
  })

  const handleFilterChange = (val: string) => { setStatusFilter(val); setCursor(0) }

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
          <Button icon="add" onClick={() => navigate('/cms/blog/new')} data-testid="blog-add-button">Add Post</Button>
        }
      />

      <div className="mb-4 flex items-center gap-3">
        <select value={statusFilter} onChange={e => handleFilterChange(e.target.value)}
          className="rounded-lg border border-[var(--outline-variant)] px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
          data-testid="blog-status-filter">
          <option value="">All Statuses</option>
          <option value="draft">Draft</option>
          <option value="published">Published</option>
        </select>
      </div>

      {allPosts.length === 0 ? (
        <Card testId="blog-empty-card">
          <EmptyState icon="article" title="No blog posts yet" description="Create your first post to get started."
            action={<Button icon="add" variant="secondary" size="sm" onClick={() => navigate('/cms/blog/new')} data-testid="blog-empty-add-button">Add Post</Button>}
            testId="blog-empty-state" />
        </Card>
      ) : (
        <>
          <Table headers={['Title', 'Slug', 'Status', 'Tags', 'Published At', 'Actions']} testId="blog-table">
            {allPosts.map((p: BlogPost) => (
              <tr key={p.id} className="hover:bg-[var(--surface-container-low)] transition-colors" data-testid={`blog-row-${p.id}`}>
                <td className="p-3 font-medium cursor-pointer" onClick={() => navigate(`/cms/blog/${p.id}/edit`)}>{p.title}</td>
                <td className="p-3 text-[var(--on-surface-variant)]">/{p.slug}</td>
                <td className="p-3"><StatusBadge status={p.status} /></td>
                <td className="p-3 text-[var(--on-surface-variant)] text-xs">{(p.tags || []).join(', ') || '—'}</td>
                <td className="p-3 text-[var(--on-surface-variant)] text-xs">{p.published_at ? new Date(p.published_at).toLocaleDateString() : '—'}</td>
                <td className="p-3 text-right">
                  <Button variant="secondary" size="sm" icon="edit" onClick={() => navigate(`/cms/blog/${p.id}/edit`)} data-testid={`blog-edit-${p.id}`} className="mr-1">Edit</Button>
                  <Button variant="secondary" size="sm" icon={p.status === 'published' ? 'unpublished' : 'publish'}
                    onClick={() => publishMutation.mutate({ id: p.id, publish: p.status !== 'published' })}
                    data-testid={`blog-publish-${p.id}`} className="mr-1">{p.status === 'published' ? 'Unpublish' : 'Publish'}</Button>
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
          onClose={() => { setConfirmDelete(null); setDeleteErr(null) }}
          onConfirm={() => deleteMutation.mutate(confirmDelete)}
          title="Delete Blog Post"
          message="This will permanently remove this blog post."
          confirmLabel="Yes, Permanently Delete"
          loading={deleteMutation.isPending}
          error={deleteErr}
          testId="blog-delete-confirm-dialog"
        />
      )}
    </main>
  )
}
