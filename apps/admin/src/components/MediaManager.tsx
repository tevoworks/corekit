import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api from '../lib/api'
import type { FileItem } from '../lib/types'
import { useState, useRef } from 'react'
import { Modal, Button, LoadingSkeleton, EmptyState } from './ui'

interface MediaManagerProps {
  open: boolean
  onClose: () => void
  onSelect: (file: { id: number; url: string; filename: string }) => void
  filterType?: string
}

export default function MediaManager({ open, onClose, onSelect, filterType = 'image' }: MediaManagerProps) {
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [cursor, setCursor] = useState(0)
  const [allFiles, setAllFiles] = useState<FileItem[]>([])
  const fileInputRef = useRef<HTMLInputElement>(null)
  const qc = useQueryClient()

  const { isLoading } = useQuery({
    queryKey: ['media-manager', cursor],
    queryFn: () => api.get(`/api/storage/files?limit=50&cursor=${cursor}`).then(r => {
      const items = Array.isArray(r.data.data) ? r.data.data : []
      if (cursor === 0) setAllFiles(items)
      else setAllFiles(prev => [...prev, ...items])
      return items
    }),
    enabled: open,
  })

  const uploadMutation = useMutation({
    mutationFn: (file: File) => {
      const fd = new FormData()
      fd.append('file', file)
      return api.post('/api/storage/upload', fd)
    },
    onSuccess: () => {
      setCursor(0)
      qc.invalidateQueries({ queryKey: ['media-manager'] })
      if (fileInputRef.current) fileInputRef.current.value = ''
    },
  })

  const images = allFiles.filter(f => f.mime_type.startsWith(filterType))

  const handleSelect = () => {
    const file = allFiles.find(f => f.id === selectedId)
    if (file) {
      onSelect({ id: file.id, url: file.url, filename: file.filename })
      onClose()
    }
  }

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return bytes + ' B'
    if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(0) + ' KB'
    return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
  }

  if (!open) return null

  return (
    <Modal onClose={onClose} title="Media Library" size="lg" testId="media-manager-modal">
      <div className="space-y-4">
        <div className="flex items-center gap-3">
          <input ref={fileInputRef} type="file" accept="image/*"
            onChange={e => { if (e.target.files?.[0]) uploadMutation.mutate(e.target.files[0]) }}
            className="hidden" data-testid="media-manager-upload-input" id="media-upload-input" />
          <label htmlFor="media-upload-input"
            className="inline-flex items-center gap-1.5 px-4 py-2 text-sm rounded-lg font-medium bg-white border border-zinc-300 text-zinc-700 hover:bg-zinc-50 transition-all cursor-pointer"
            data-testid="media-manager-upload-button">
            {uploadMutation.isPending ? (
              <span className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" />
            ) : (
              <span className="material-symbols-outlined text-lg">upload</span>
            )}
            {uploadMutation.isPending ? 'Uploading...' : 'Upload'}
          </label>
          <span className="text-xs text-zinc-400">{images.length} image(s)</span>
        </div>

        {isLoading ? (
          <LoadingSkeleton rows={6} testId="media-manager-loading" />
        ) : images.length === 0 ? (
          <EmptyState icon="image" title="No images yet" description="Upload images to use in your content." testId="media-manager-empty" />
        ) : (
          <div className="grid grid-cols-3 sm:grid-cols-4 gap-3 max-h-80 overflow-y-auto">
            {images.map((f) => (
              <div key={f.id}
                onClick={() => setSelectedId(f.id)}
                className={`relative rounded-lg border-2 overflow-hidden cursor-pointer transition-all hover:shadow-md
                  ${selectedId === f.id ? 'border-blue-500 ring-2 ring-blue-200' : 'border-zinc-200'}`}
                data-testid={`media-manager-item-${f.id}`}>
                <div className="aspect-square bg-zinc-100 flex items-center justify-center overflow-hidden">
                  <img src={f.url} alt={f.filename} className="w-full h-full object-cover"
                    onError={(e) => { (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><rect width="24" height="24" fill="%23eee"/><text x="12" y="14" text-anchor="middle" font-size="10" fill="%23999">IMG</text></svg>' }} />
                </div>
                <div className="p-1.5 text-xs truncate text-zinc-600 bg-white">{f.filename}</div>
                <div className="px-1.5 pb-1 text-[10px] text-zinc-400 bg-white">{formatSize(f.size_bytes)}</div>
              </div>
            ))}
          </div>
        )}

        {allFiles.length >= 50 && (
          <div className="text-center">
            <Button variant="ghost" size="sm" onClick={() => setCursor(c => c + 50)} data-testid="media-manager-load-more">Load More</Button>
          </div>
        )}
      </div>
      <div className="flex gap-2 justify-end mt-4 pt-4 border-t border-zinc-200">
        <Button variant="ghost" onClick={onClose} data-testid="media-manager-cancel">Cancel</Button>
        <Button onClick={handleSelect} disabled={!selectedId} data-testid="media-manager-select">Select</Button>
      </div>
    </Modal>
  )
}
