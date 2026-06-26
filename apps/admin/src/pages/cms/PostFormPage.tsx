import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import api, { getApiError } from '../../lib/api'
import type { BlogPost } from '../../lib/types'
import { useState, useRef, useEffect, useCallback } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Card, PageHeader, Button, Input, StatusBadge } from '../../components/ui'
import RichTextEditor from '../../components/RichTextEditor'
import { getDraftKey, saveDraft, loadDraft, clearDraft } from '../../lib/draft'

function slugify(text: string): string {
  return text.toLowerCase().trim().replace(/[^\w\s-]/g, '').replace(/[\s_]+/g, '-').replace(/-+/g, '-')
}

export default function PostFormPage() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const isEdit = !!id

  const [title, setTitle] = useState('')
  const [slug, setSlug] = useState('')
  const [content, setContent] = useState('')
  const [excerpt, setExcerpt] = useState('')
  const [tags, setTags] = useState('')
  const [metaTitle, setMetaTitle] = useState('')
  const [metaDescription, setMetaDescription] = useState('')
  const [ogImage, setOgImage] = useState('')
  const [featuredImageId, setFeaturedImageId] = useState<number | null>(null)
  const [featuredImageUrl, setFeaturedImageUrl] = useState('')
  const [slugConflict, setSlugConflict] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [wasAutoSlugged, setWasAutoSlugged] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [draftBanner, setDraftBanner] = useState<{ show: boolean; savedAt: string }>({ show: false, savedAt: '' })
  const [isDirty, setIsDirty] = useState(false)
  const slugCheckTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const draftTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const draftKey = getDraftKey('post', id)
  const hasInitialized = useRef(false)

  const { data, isLoading, isError } = useQuery({
    queryKey: ['CMS', 'post', id],
    queryFn: () => api.get(`/api/cms/posts/${id}`).then(r => r.data.data as BlogPost),
    enabled: isEdit,
  })

  useEffect(() => {
    if (!data) return
    setTitle(data.title)
    setSlug(data.slug)
    setContent(data.content)
    setExcerpt(data.excerpt || '')
    setTags((data.tags || []).join(', '))
    setMetaTitle(data.meta_title || '')
    setMetaDescription(data.meta_description || '')
    setOgImage(data.og_image || '')
    setFeaturedImageId(data.featured_image_id ?? null)
    setFeaturedImageUrl(data.featured_image || '')
    setWasAutoSlugged(false)
    hasInitialized.current = true
  }, [data])

  useEffect(() => {
    if (isError) navigate('/cms/blog')
  }, [isError])

  useEffect(() => {
    if (!isEdit && !hasInitialized.current) {
      const draft = loadDraft(draftKey)
      if (draft) {
        setDraftBanner({ show: true, savedAt: draft.savedAt })
      }
      hasInitialized.current = true
    }
  }, [isEdit, draftKey])

  useEffect(() => {
    if (!isDirty || isEdit) return
    if (draftTimer.current) clearTimeout(draftTimer.current)
    draftTimer.current = setTimeout(() => {
      saveDraft(draftKey, {
        title, slug, content, excerpt, tags,
        meta_title: metaTitle,
        meta_description: metaDescription, og_image: ogImage,
        featured_image_id: featuredImageId, featured_image: featuredImageUrl,
      })
    }, 2000)
    return () => { if (draftTimer.current) clearTimeout(draftTimer.current) }
  }, [isDirty, isEdit, draftKey, title, slug, content, excerpt, tags, metaTitle, metaDescription, ogImage, featuredImageId, featuredImageUrl])

  useEffect(() => {
    if (!isDirty) return
    const handler = (e: BeforeUnloadEvent) => { e.preventDefault() }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [isDirty])

  const markDirty = useCallback(() => { if (!isDirty) setIsDirty(true) }, [isDirty])

  const restoreDraft = () => {
    const draft = loadDraft(draftKey)
    if (!draft) return
    const d = draft.body
    setTitle(d.title || '')
    setSlug(d.slug || '')
    setContent(d.content || '')
    setExcerpt(d.excerpt || '')
    setTags(d.tags || '')
    setMetaTitle(d.meta_title || '')
    setMetaDescription(d.meta_description || '')
    setOgImage(d.og_image || '')
    setFeaturedImageId(d.featured_image_id ?? null)
    setFeaturedImageUrl(d.featured_image || '')
    setDraftBanner({ show: false, savedAt: '' })
  }

  const discardDraft = () => {
    clearDraft(draftKey)
    setDraftBanner({ show: false, savedAt: '' })
  }

  const checkSlug = (val: string) => {
    if (slugCheckTimer.current) clearTimeout(slugCheckTimer.current)
    if (!val.trim()) { setSlugConflict(false); return }
    slugCheckTimer.current = setTimeout(async () => {
      try {
        const res = await api.get(`/api/cms/check-slug?slug=${encodeURIComponent(val)}&exclude_id=${id || 0}`)
        setSlugConflict(res.data.data?.taken === true)
      } catch { setSlugConflict(false) }
    }, 500)
  }

  const handleTitleChange = (val: string) => {
    setTitle(val); markDirty()
    if (wasAutoSlugged) setSlug(slugify(val))
  }

  const handleSlugChange = (val: string) => {
    setWasAutoSlugged(false); setSlug(val); markDirty()
  }

  const handleSlugBlur = () => { checkSlug(slug) }

  const handleUploadImage = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setIsUploading(true)
    try {
      const formData = new FormData()
      formData.append('file', file)
      const res = await api.post('/api/storage/upload', formData, { headers: { 'Content-Type': 'multipart/form-data' } })
      const fd = res.data.data
      setFeaturedImageId(fd.id)
      setFeaturedImageUrl(fd.url || fd.storage_path)
      markDirty()
    } catch (err: any) { setError(getApiError(err)) }
    finally { setIsUploading(false) }
  }

  const handleRemoveImage = () => {
    setFeaturedImageId(null)
    setFeaturedImageUrl('')
    markDirty()
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  const createMutation = useMutation({
    mutationFn: (body: any) => api.post('/api/cms/posts', body),
    onSuccess: () => { clearDraft(draftKey); qc.invalidateQueries({ queryKey: ['CMS', 'blog'] }); navigate('/cms/blog') },
    onError: (err: any) => setError(getApiError(err)),
  })

  const updateMutation = useMutation({
    mutationFn: (body: any) => api.put(`/api/cms/posts/${id}`, body),
    onSuccess: () => { clearDraft(draftKey); qc.invalidateQueries({ queryKey: ['CMS', 'blog'] }); navigate('/cms/blog') },
    onError: (err: any) => setError(getApiError(err)),
  })

  const saving = createMutation.isPending || updateMutation.isPending

  const handleSubmit = () => {
    const body = {
      title, slug, content, excerpt,
      meta_title: metaTitle,
      meta_description: metaDescription,
      og_image: ogImage,
      featured_image_id: featuredImageId,
      featured_image: featuredImageUrl,
      tags: tags.split(',').map(t => t.trim()).filter(Boolean),
    }
    if (isEdit) updateMutation.mutate(body)
    else createMutation.mutate(body)
  }

  if (isEdit && isLoading) {
    return (
      <main className="content-canvas animate-fade-in">
        <PageHeader title="Loading..." description="Fetching post data" />
        <Card testId="post-form-loading-card"><div className="p-8 text-center text-[var(--on-surface-variant)]">Loading...</div></Card>
      </main>
    )
  }

  return (
    <main className="content-canvas animate-fade-in">
      <PageHeader
        title={isEdit ? 'Edit Post' : 'Create Post'}
        description={isEdit ? `Editing: ${title}` : 'Create a new blog post'}
        action={
          <Button variant="ghost" icon="arrow_back" onClick={() => navigate('/cms/blog')} data-testid="post-form-back-button">
            Back to Blog
          </Button>
        }
      />

      {draftBanner.show && (
        <div className="mb-4 p-3 rounded-lg bg-amber-50 border border-amber-200 text-amber-800 text-sm flex items-center gap-2" data-testid="post-form-draft-banner">
          <span className="material-symbols-outlined text-base">description</span>
          <span className="flex-1">
            You have an unsaved draft from {new Date(draftBanner.savedAt).toLocaleTimeString()}.
          </span>
          <button onClick={restoreDraft} className="text-sm font-medium underline hover:no-underline" data-testid="post-form-draft-restore">Restore</button>
          <button onClick={discardDraft} className="text-sm font-medium underline hover:no-underline" data-testid="post-form-draft-discard">Discard</button>
        </div>
      )}

      {error && (
        <div className="mb-4 p-3 rounded-lg bg-[var(--danger-bg)] text-[var(--danger-text)] text-sm flex items-center gap-2" data-testid="post-form-error">
          <span className="material-symbols-outlined text-base">error</span>
          <span className="flex-1">{error}</span>
          <button onClick={() => setError(null)} className="text-[var(--danger-text)] hover:opacity-70" data-testid="post-form-error-dismiss">
            <span className="material-symbols-outlined text-base">close</span>
          </button>
        </div>
      )}

      <Card testId="post-form-card" padding={false}>
        <div className="p-6 space-y-6">
          <Input label="Title *" value={title} onChange={e => handleTitleChange(e.target.value)} required data-testid="post-form-title" />
          <div>
            <Input
              label="Slug"
              value={slug}
              onChange={e => handleSlugChange(e.target.value)}
              onBlur={handleSlugBlur}
              data-testid="post-form-slug"
              placeholder="auto-generated from title"
            />
            {slugConflict && (
              <p className="mt-1 text-xs text-amber-600 flex items-center gap-1" data-testid="post-form-slug-warning">
                <span className="material-symbols-outlined text-sm">warning</span>
                This slug is already taken. Consider using a different one.
              </p>
            )}
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--on-surface)] mb-1">Content</label>
            <RichTextEditor content={content} onChange={v => { setContent(v); markDirty() }} testId="post-form-content-editor" />
          </div>
          <div>
            <label className="block text-sm font-medium text-[var(--on-surface)] mb-1">Excerpt</label>
            <textarea value={excerpt} onChange={e => { setExcerpt(e.target.value); markDirty() }} rows={3}
              className="w-full px-3 py-3 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent transition-all"
              data-testid="post-form-excerpt" />
          </div>
          <Input label="Tags (comma separated)" value={tags} onChange={e => { setTags(e.target.value); markDirty() }} data-testid="post-form-tags" />

          <div className="border-t border-[var(--outline-variant)] pt-6">
            <h3 className="text-sm font-semibold text-[var(--on-surface)] mb-3">SEO Settings</h3>
            <div className="space-y-3">
              <Input label="Meta Title" value={metaTitle} onChange={e => { setMetaTitle(e.target.value); markDirty() }} data-testid="post-form-meta-title" />
              <div>
                <label className="block text-sm font-medium text-[var(--on-surface)] mb-1">Meta Description</label>
                <textarea value={metaDescription} onChange={e => { setMetaDescription(e.target.value); markDirty() }} rows={3}
                  className="w-full px-3 py-3 rounded-lg border border-[var(--outline-variant)] text-sm bg-white focus:outline-none focus:ring-2 focus:ring-[var(--primary)] focus:border-transparent transition-all"
                  data-testid="post-form-meta-description" />
              </div>
              <Input label="OG Image URL" value={ogImage} onChange={e => { setOgImage(e.target.value); markDirty() }} data-testid="post-form-og-image" placeholder="https://..." />
            </div>
          </div>

          <div className="border-t border-[var(--outline-variant)] pt-6">
            <h3 className="text-sm font-semibold text-[var(--on-surface)] mb-3">Featured Image</h3>
            <div className="flex items-center gap-3">
              <input ref={fileInputRef} type="file" accept="image/*" onChange={handleUploadImage} className="hidden" data-testid="post-form-image-input" id="post-featured-image-input" />
              <label htmlFor="post-featured-image-input"
                className="inline-flex items-center gap-1.5 px-4 py-2 text-sm rounded-lg font-medium bg-white border border-[var(--outline-variant)] text-[var(--on-surface)] hover:bg-[var(--surface-container)] transition-all cursor-pointer"
                data-testid="post-form-image-upload">
                {isUploading ? <span className="w-4 h-4 border-2 border-current border-t-transparent rounded-full animate-spin" /> : <span className="material-symbols-outlined text-lg">upload</span>}
                {isUploading ? 'Uploading...' : 'Upload Image'}
              </label>
              {featuredImageUrl && (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-[var(--on-surface-variant)] truncate max-w-[120px]" data-testid="post-form-image-filename">{featuredImageUrl.split('/').pop()}</span>
                  <button type="button" onClick={handleRemoveImage} className="text-xs text-red-500 hover:text-red-700" data-testid="post-form-image-remove">Remove</button>
                </div>
              )}
            </div>
            {featuredImageUrl && (
              <div className="mt-2">
                <img src={featuredImageUrl} alt="Preview" className="h-24 w-auto rounded border border-[var(--outline-variant)] object-cover" data-testid="post-form-image-preview" />
              </div>
            )}
          </div>
        </div>
      </Card>

      <div className="mt-6 flex items-center justify-between">
        <span className="text-xs text-[var(--on-surface-variant)]" data-testid="post-form-draft-indicator">
          {isDirty && !saving ? 'Unsaved changes' : ''}
        </span>
        <div className="flex gap-2">
          <Button variant="ghost" onClick={() => navigate('/cms/blog')} data-testid="post-form-cancel">Cancel</Button>
          <Button onClick={handleSubmit} loading={saving} disabled={!title} data-testid="post-form-submit">
            {isEdit ? 'Update Post' : 'Create Post'}
          </Button>
        </div>
      </div>
    </main>
  )
}
