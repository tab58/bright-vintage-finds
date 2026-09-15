// API client for the admin inventory endpoints. The base is same-origin:
// `/env.js` (dev: Vite proxy, prod: Caddy) defines window.BACKEND_API when the
// API is served from a different origin; otherwise requests go to the app
// origin itself, which in dev is the Vite proxy to main-api.
const base: string =
  (typeof window !== 'undefined' && (window as any).BACKEND_API) || ''

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${base}${path}`, {
    headers:
      init?.body && !(init.body instanceof FormData)
        ? { 'Content-Type': 'application/json' }
        : undefined,
    ...init,
  })
  if (!res.ok) {
    let detail = res.statusText
    try {
      const body = await res.json()
      if (body?.detail) detail = body.detail
    } catch {
      // non-JSON error body
    }
    throw new Error(`${res.status}: ${detail}`)
  }
  if (res.status === 204) return undefined as T
  return res.json()
}

// The API returns bare JSON; some endpoints were assumed to wrap it in { body }.
// Accept either shape so a single response style change can't blank a page.
function unwrap<T>(res: T | { body: T }): T {
  return res && typeof res === 'object' && 'body' in (res as object)
    ? (res as { body: T }).body
    : (res as T)
}

export interface SellingPlace {
  id: string
  name: string
  is_builtin: boolean
}

export interface Label {
  id: string
  name: string
}

export interface ItemImage {
  id: string
  url: string
  display_order: number
}

export type ItemStatus = 'draft' | 'listed' | 'sold' | 'archived'

export interface Item {
  id: string
  name: string
  description?: string
  category?: string
  condition?: string
  status: ItemStatus
  acquisition_cost_cents?: number
  purchased_at?: string
  listing_price_cents?: number
  length?: number
  width?: number
  height?: number
  measurement_unit: 'inch' | 'cm'
  extra_measurements?: string
  weight_lbs?: number
  weight_oz?: number
  notes?: string
  whatnot_number?: string
  selling_places: string[]
  labels: string[]
  selling_place_ids: string[]
  label_ids: string[]
  image_count: number
  listed_at?: string
  first_listed_at?: string
  sold_price_cents?: number
  sold_at?: string
  sold_place?: string
  sold_place_id?: string
  created_at: string
  updated_at: string
}

export interface ItemBody {
  name: string
  description?: string
  acquisition_cost_cents?: number
  purchased_at?: string
  length?: number
  width?: number
  height?: number
  measurement_unit?: 'inch' | 'cm'
  extra_measurements?: string
  weight_lbs?: number
  weight_oz?: number
  notes?: string
  whatnot_number?: string
  selling_place_ids?: string[]
  label_ids?: string[]
  status?: ItemStatus
  sold_price_cents?: number
  sold_at?: string
  sold_place_id?: string
}

export interface ListFilters {
  query?: string
  place_id?: string
  label_id?: string
  whatnot_number?: string
  status?: string
}

export async function listSellingPlaces(): Promise<SellingPlace[]> {
  const out = await request<{ body: SellingPlace[] }>('/admin/selling-places')
  return unwrap(out)
}

export function createSellingPlace(name: string): Promise<SellingPlace> {
  return request<SellingPlace>('/admin/selling-places', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}

export async function listLabels(): Promise<Label[]> {
  const out = await request<{ body: Label[] }>('/admin/labels')
  return unwrap(out)
}

export function createLabel(name: string): Promise<Label> {
  return request<Label>('/admin/labels', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}

export interface ItemFilters {
  query?: string
  place_id?: string
  label_id?: string
  whatnot_number?: string
  status?: string
}

export async function listItems(filters: ItemFilters = {}): Promise<Item[]> {
  const params = new URLSearchParams()
  for (const [k, v] of Object.entries(filters)) {
    if (v) params.set(k, v)
  }
  const qs = params.toString()
  const out = await request<{ body: Item[] }>(`/admin/items${qs ? `?${qs}` : ''}`)
  return unwrap(out)
}

export async function getItem(id: string): Promise<Item> {
  const out = await request<{ body: Item }>(`/admin/items/${id}`)
  return unwrap(out)
}

export function createItem(body: ItemBody): Promise<Item> {
  return request<Item>('/admin/items', {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function updateItem(id: string, body: ItemBody): Promise<Item> {
  return request<Item>(`/admin/items/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  })
}

export interface MarkSoldBody {
  sold_at: string
  sold_price_cents: number
  sold_place_id: string
}

export function markItemSold(id: string, body: MarkSoldBody): Promise<Item> {
  return request<Item>(`/admin/items/${id}/sold`, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export async function listItemImages(itemId: string): Promise<ItemImage[]> {
  const out = await request<{ body: ItemImage[] }>(`/admin/items/${itemId}/images`)
  return unwrap(out)
}

export async function uploadImage(
  itemId: string,
  file: File,
): Promise<ItemImage> {
  const form = new FormData()
  form.append('file', file)
  const out = await request<{ body: ItemImage }>(
    `/admin/items/${itemId}/images`,
    { method: 'POST', body: form },
  )
  return unwrap(out)
}