// API client for the unauthenticated /public catalog endpoints the shop front
// page reads. Separate from client.ts so nothing admin-only leaks into the
// public page's bundle of concerns.
import { request, unwrap } from './client';

export interface PublicItem {
  id: string;
  name: string;
  description?: string;
  category?: string;
  condition?: string;
  listing_price_cents?: number;
  length?: number;
  width?: number;
  height?: number;
  measurement_unit: 'inch' | 'cm';
  extra_measurements?: string;
  labels: string[];
  image_count: number;
  cover_image_url?: string;
}

export interface PublicImage {
  id: string;
  url: string;
  display_order: number;
}

export interface PublicItemsPage {
  items: PublicItem[];
  // Absent on the last page.
  next_cursor?: string;
}

export interface PublicFilters {
  categories: string[];
  labels: string[];
}

export interface PublicItemQuery {
  query?: string;
  category?: string;
  label?: string;
  cursor?: string;
  limit?: number;
}

export async function listPublicItems(
  params: PublicItemQuery = {},
): Promise<PublicItemsPage> {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value) search.set(key, String(value));
  }
  const qs = search.toString();
  const out = await request<PublicItemsPage | { body: PublicItemsPage }>(
    `/public/items${qs ? `?${qs}` : ''}`,
  );
  const page = unwrap(out);
  return { items: page.items ?? [], next_cursor: page.next_cursor };
}

export async function getPublicFilters(): Promise<PublicFilters> {
  const out = await request<PublicFilters | { body: PublicFilters }>(
    '/public/filters',
  );
  const filters = unwrap(out);
  return { categories: filters.categories ?? [], labels: filters.labels ?? [] };
}

export async function listPublicItemImages(
  itemId: string,
): Promise<PublicImage[]> {
  const out = await request<PublicImage[] | { body: PublicImage[] }>(
    `/public/items/${itemId}/images`,
  );
  return unwrap(out) ?? [];
}
