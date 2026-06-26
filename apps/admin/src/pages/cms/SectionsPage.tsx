import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { PageSection } from '../../lib/types'
import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, PageHeader, Table, Button, Modal, Input, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

const SECTION_TYPES = [
  { value: 'hero', label: 'Hero' },
  { value: 'features', label: 'Features' },
  { value: 'testimonials', label: 'Testimonials' },
  { value: 'cta', label: 'CTA' },
  { value: 'pricing', label: 'Pricing' },
  { value: 'faq', label: 'FAQ' },
]

export default function SectionsPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const { pageId } = useParams<{ pageId: string }>()
  const [showForm, setShowForm] = useState(false)
  const [editing, setEditing] = useState<PageSection | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  const { data: sections = [], isLoading } = useQuery({
    queryKey: ['CMS', 'sections', pageId],
    queryFn: () => api.get(`/api/cms/pages/${pageId}/sections`).then(r => {
      const d = r.data.data
      return Array.isArray(d) ? d : []
    }),
    enabled: !!pageId,
  })

  const createMutation = useMutation({
    mutationFn: (body: any) => api.post(`/api/cms/pages/${pageId}/sections`, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'sections', pageId] }); setShowForm(false) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to create section'),
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, ...body }: any) => api.put(`/api/cms/pages/${pageId}/sections/${id}`, body),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'sections', pageId] }); setEditing(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to update section'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/cms/pages/${pageId}/sections/${id}`),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['CMS', 'sections', pageId] }); setConfirmDelete(null) },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete section'),
  })

  if (isLoading) return (
    <main className="content-canvas animate-fade-in">
      <PageHeader title="Page Sections" description="Manage sections for this page" />
      <Card testId="sections-loading-card"><LoadingSkeleton rows={4} testId="sections-loading" /></Card>
    </main>
  )

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="Page Sections"
        description="Manage sections for this page"
        action={
          <div className="flex gap-2">
            <Button variant="ghost" icon="arrow_back" onClick={() => navigate('/cms/pages')} data-testid="sections-back-button">Back to Pages</Button>
            <Button icon="add" onClick={() => setShowForm(true)} data-testid="sections-add-button">Add Section</Button>
          </div>
        }
      />

      {showForm && (
        <Modal title="Create Section" onClose={() => { setShowForm(false); setError(null) }} testId="sections-create-modal">
          <SectionFormContent
            onSave={(body) => createMutation.mutate(body)}
            onCancel={() => { setShowForm(false); setError(null) }}
            saving={createMutation.isPending}
            error={error}
            onClearError={() => setError(null)}
          />
        </Modal>
      )}
      {editing && (
        <Modal title="Edit Section" onClose={() => { setEditing(null); setError(null) }} testId="sections-edit-modal">
          <SectionFormContent
            section={editing}
            onSave={(body) => updateMutation.mutate({ id: editing.id, ...body })}
            onCancel={() => { setEditing(null); setError(null) }}
            saving={updateMutation.isPending}
            error={error}
            onClearError={() => setError(null)}
          />
        </Modal>
      )}

      {sections.length === 0 ? (
        <Card testId="sections-empty-card"><EmptyState icon="dashboard" title="No sections yet" description="Add your first section to this page." action={<Button icon="add" variant="secondary" size="sm" onClick={() => setShowForm(true)} data-testid="sections-empty-add-button">Add Section</Button>} testId="sections-empty-state" /></Card>
      ) : (
        <Table headers={['Type', 'Title', 'Sort Order', 'Actions']} testId="sections-table">
          {(sections as PageSection[]).sort((a, b) => a.sort_order - b.sort_order).map((s: PageSection) => (
            <tr key={s.id} className="hover:bg-[var(--surface-container-low)] transition-colors" data-testid={`sections-row-${s.id}`}>
              <td className="p-3"><span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-[var(--info-bg)] text-[var(--info-text)]">{s.type}</span></td>
              <td className="p-3 font-medium">{s.title}</td>
              <td className="p-3 text-[var(--on-surface-variant)]">{s.sort_order}</td>
              <td className="p-3 text-right">
                <Button variant="secondary" size="sm" icon="edit" onClick={() => setEditing(s)} data-testid={`sections-edit-${s.id}`} className="mr-1">Edit</Button>
                <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(s.id)} data-testid={`sections-delete-${s.id}`}>Delete</Button>
              </td>
            </tr>
          ))}
        </Table>
      )}

      {confirmDelete && (
        <ConfirmDialog
          open={!!confirmDelete}
          onClose={() => { setConfirmDelete(null); setError(null) }}
          onConfirm={() => deleteMutation.mutate(confirmDelete)}
          title="Delete Section"
          message="This will permanently remove this section."
          confirmLabel="Yes, Permanently Delete"
          loading={deleteMutation.isPending}
          error={error}
          testId="sections-delete-confirm-dialog"
        />
      )}
    </main>
  )
}

function SectionFormContent({ section, onSave, onCancel, saving, error, onClearError }: {
  section?: PageSection | null
  onSave: (body: any) => void
  onCancel: () => void
  saving?: boolean
  error?: string | null
  onClearError?: () => void
}) {
  const [type, setType] = useState(section?.type || 'hero')
  const [title, setTitle] = useState(section?.title || '')
  const [content, setContent] = useState(section?.content ? JSON.stringify(section.content, null, 2) : '')
  const [sortOrder, setSortOrder] = useState(String(section?.sort_order ?? 0))

  return (
    <div>
      {error && (
        <div className="mb-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm flex items-center gap-2" data-testid="sections-form-error">
          <span className="material-symbols-outlined text-base">error</span>
          <span className="flex-1">{error}</span>
          <button onClick={onClearError} className="text-[var(--danger-text)] hover:opacity-70">
            <span className="material-symbols-outlined text-base">close</span>
          </button>
        </div>
      )}
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-[var(--on-surface)] mb-1">Type</label>
          <select
            value={type}
            onChange={e => setType(e.target.value)}
            className="w-full rounded-lg border border-[var(--outline-variant)] px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent"
            data-testid="sections-form-type"
          >
            {SECTION_TYPES.map(t => (
              <option key={t.value} value={t.value}>{t.label}</option>
            ))}
          </select>
        </div>
        <Input label="Title *" value={title} onChange={e => setTitle(e.target.value)} required data-testid="sections-form-title" />
        <div>
          <label className="block text-sm font-medium text-[var(--on-surface)] mb-1">Content (JSON)</label>
          <textarea value={content} onChange={e => setContent(e.target.value)} rows={6} className="w-full px-3 py-3 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent transition-all" data-testid="sections-form-content" />
        </div>
        <Input label="Sort Order" type="number" value={sortOrder} onChange={e => setSortOrder(e.target.value)} data-testid="sections-form-sort" />
      </div>
      <div className="flex gap-2 justify-end mt-6 pt-4 border-t border-[var(--outline-variant)]">
        <Button variant="ghost" onClick={onCancel} data-testid="sections-form-cancel">Cancel</Button>
        <Button onClick={() => {
          let parsed = {}
          try { parsed = JSON.parse(content) } catch {}
          onSave({ type, title, content: parsed, sort_order: Number(sortOrder) })
        }} loading={saving} disabled={!title} data-testid="sections-form-submit">{section ? 'Update' : 'Create'}</Button>
      </div>
    </div>
  )
}
