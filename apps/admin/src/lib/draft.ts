export function getDraftKey(type: 'page' | 'post', id?: number | string): string {
  return `cms_draft_${type}_${id || 'new'}`
}

export interface DraftData {
  body: Record<string, any>
  savedAt: string
}

export function saveDraft(key: string, body: Record<string, any>): void {
  try {
    const data: DraftData = { body, savedAt: new Date().toISOString() }
    localStorage.setItem(key, JSON.stringify(data))
  } catch {
    // localStorage full or unavailable — silently ignore
  }
}

export function loadDraft(key: string): DraftData | null {
  try {
    const raw = localStorage.getItem(key)
    if (!raw) return null
    return JSON.parse(raw) as DraftData
  } catch {
    return null
  }
}

export function clearDraft(key: string): void {
  try {
    localStorage.removeItem(key)
  } catch {
    // ignore
  }
}

export function hasDraft(key: string): boolean {
  return loadDraft(key) !== null
}
