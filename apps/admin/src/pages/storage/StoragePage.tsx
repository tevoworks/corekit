import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../../lib/api'
import type { FileItem } from '../../lib/types'
import { useRef, useState } from 'react'
import { Card, PageHeader, Table, Button, Badge, StatusBadge, EmptyState, LoadingSkeleton, ConfirmDialog } from '../../components/ui'

export default function StoragePage() {
  const qc = useQueryClient()
  const fileRef = useRef<HTMLInputElement>(null)
  const [error, setError] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null)

  const { data: files = [], isLoading } = useQuery({
    queryKey: ['files'],
    queryFn: () => api.get('/api/storage/files').then(r => Array.isArray(r.data.data) ? r.data.data : []),
  })

  const uploadMutation = useMutation({
    mutationFn: (file: File) => {
      const fd = new FormData()
      fd.append('file', file)
      return api.post('/api/storage/upload', fd)
    },
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['files'] }); if (fileRef.current) fileRef.current.value = '' },
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to upload file'),
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/storage/files/${id}`),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['files'] }),
    onError: (err: any) => setError(err.response?.data?.error?.message || 'Failed to delete file'),
  })

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  }

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title="Storage"
        description="Uploaded files"
        action={
          <>
            <input ref={fileRef} type="file" className="hidden" onChange={e => { if (e.target.files?.[0]) uploadMutation.mutate(e.target.files[0]) }} data-testid="storage-file-input" />
            <Button icon="upload" onClick={() => fileRef.current?.click()} data-testid="storage-upload-button">Upload File</Button>
          </>
        }
      />

      {isLoading ? (
        <Card testId="storage-loading-card"><LoadingSkeleton rows={5} testId="storage-loading" /></Card>
      ) : files.length === 0 ? (
        <Card testId="storage-empty-card"><EmptyState icon="folder" title="No files uploaded yet" description="Upload a file using the button above." action={<Button icon="upload" variant="secondary" size="sm" onClick={() => fileRef.current?.click()} data-testid="storage-empty-upload-button">Upload File</Button>} testId="storage-empty-state" /></Card>
      ) : (
        <Table headers={['Name', 'Type', 'Size', 'Public', 'Uploaded', 'Actions']} testId="storage-table">
          {files.map((f: FileItem) => (
              <tr key={f.id} className="hover:bg-[var(--surface-container-low)] transition-colors" data-testid={`storage-row-${f.id}`}>
                <td className="p-3 font-medium">{f.filename}</td>
                <td className="p-3 text-xs text-[var(--on-surface-variant)]"><Badge variant="neutral">{f.mime_type}</Badge></td>
                <td className="p-3 text-[var(--on-surface-variant)] text-xs font-mono">{formatSize(f.size_bytes)}</td>
                <td className="p-3"><StatusBadge status={f.is_public ? 'ACTIVE' : 'HALTED'} /></td>
                <td className="p-3 text-xs text-[var(--on-surface-variant)]">{new Date(f.created_at).toLocaleDateString()}</td>
                <td className="p-3 text-right">
                  <a href={`${import.meta.env.VITE_API_URL || ''}/api/storage/files/${f.id}`} target="_blank" rel="noopener noreferrer" data-testid={`storage-download-${f.id}`}>
                    <Button variant="ghost" size="sm" icon="download" className="mr-1">Download</Button>
                  </a>
                  <Button variant="danger" size="sm" icon="delete" onClick={() => setConfirmDelete(f.id)} data-testid={`storage-delete-${f.id}`}>Delete</Button>
                </td>
              </tr>
          ))}
        </Table>
      )}

      <ConfirmDialog
        open={confirmDelete !== null}
        onClose={() => setConfirmDelete(null)}
        onConfirm={() => deleteMutation.mutate(confirmDelete!)}
        title="Delete File"
        message="Are you sure you want to delete this file? This action cannot be undone."
        confirmLabel="Delete"
        confirmVariant="danger"
        loading={deleteMutation.isPending}
        error={error}
        testId="storage-delete-modal"
      />
    </main>
  )
}
