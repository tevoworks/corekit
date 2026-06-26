import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api, { getApiError } from '../../lib/api'
import type { Page } from '../../lib/types'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card, PageHeader, Table, Button, StatusBadge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

export default function PagesPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [statusFilter, setStatusFilter] = useState('')
  const [cursor, setCursor] = useState(0)
  const [allPages, setAllPages] = useState<Page[]>([])
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)
  const [deleteErr, setDeleteErr] = useState<string | null>(null)

  const { isLoading } = useQuery({
    queryKey: ['CMS', 'pages', statusFilter, cursor],
    queryFn: () => api.get(`/api/cms/pages?status=${statusFilter}&limit=50&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data : []
      if (cursor === 0) setAllPages(items)
      else setAllPages(prev => [...prev, ...items])
      return { items, next_cursor: r.data.meta?.next_cursor || 0 }
    }),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/cms/pages/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'pages'] }); setCursor(0); setConfirmDelete(null) },
    onError: (err: any) => { setDeleteErr(getApiError(err)) },
  })

  const publishMutation = useMutation({
    mutationFn: ({ id, publish }: { id: number; publish: boolean }) => api.post(`/api/cms/pages/${id}/${publish ? 'publish' : 'unpublish'}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'pages'] }) },
    onError: (err: any) => { setDeleteErr(getApiError(err)) },
  })

  const handleFilterChange = (val: string) => { setStatusFilter(val); setCursor(0) }

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
          <Button icon="add" onClick={() => navigate('/cms/pages/new')} data-testid="pages-add-button">Add Page</Button>
        }
      />

      <div className="mb-4 flex items-center gap-3">
        <select value={statusFilter} onChange={e => handleFilterChange(e.target.value)}
          className="rounded-lg border border-[var(--outline-variant)] px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
          data-testid="pages-status-filter">
          <option value="">All Statuses</option>
          <option value="draft">Draft</option>
          <option value="published">Published</option>
        </select>
      </div>

      {allPages.length === 0 ? (
        <Card testId="pages-empty-card">
          <EmptyState icon="description" title="No pages yet" description="Create your first page to get started."
            action={<Button icon="add" variant="secondary" size="sm" onClick={() => navigate('/cms/pages/new')} data-testid="pages-empty-add-button">Add Page</Button>}
            testId="pages-empty-state" />
        </Card>
      ) : (
        <>
          <Table headers={['Title', 'Slug', 'Status', 'Updated At', 'Actions']} testId="pages-table">
            {allPages.map((p: Page) => (
              <tr key={p.id} className="hover:bg-[var(--surface-container-low)] transition-colors" data-testid={`pages-row-${p.id}`}>
                <td className="p-3 font-medium cursor-pointer" onClick={() => navigate(`/cms/pages/${p.id}/edit`)}>{p.title}</td>
                <td className="p-3 text-[var(--on-surface-variant)]">/{p.slug}</td>
                <td className="p-3"><StatusBadge status={p.status} /></td>
                <td className="p-3 text-[var(--on-surface-variant)] text-xs">{new Date(p.updated_at).toLocaleDateString()}</td>
                <td className="p-3 text-right">
                  <Button variant="secondary" size="sm" icon="edit" onClick={() => navigate(`/cms/pages/${p.id}/edit`)} data-testid={`pages-edit-${p.id}`} className="mr-1">Edit</Button>
                  <Button variant="secondary" size="sm" icon={p.status === 'published' ? 'unpublished' : 'publish'}
                    onClick={() => publishMutation.mutate({ id: p.id, publish: p.status !== 'published' })}
                    data-testid={`pages-publish-${p.id}`} className="mr-1">{p.status === 'published' ? 'Unpublish' : 'Publish'}</Button>
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
          onClose={() => { setConfirmDelete(null); setDeleteErr(null) }}
          onConfirm={() => deleteMutation.mutate(confirmDelete)}
          title="Delete Page"
          message="This will permanently remove this page and all its sections."
          confirmLabel="Yes, Permanently Delete"
          loading={deleteMutation.isPending}
          error={deleteErr}
          testId="pages-delete-confirm-dialog"
        />
      )}
    </main>
  )
}
