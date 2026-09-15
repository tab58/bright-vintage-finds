// Display helpers shared by the shop cards and the lightbox.
import type { PublicItem } from '@/api/public';

// formatPrice renders cents as dollars, dropping ".00" on whole amounts.
export function formatPrice(cents?: number): string | null {
  if (cents == null) return null;
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: cents % 100 === 0 ? 0 : 2,
  }).format(cents / 100);
}

// formatDimensions renders whichever of L/W/H the owner filled in, e.g.
// `12 × 8 × 3 in`, and falls back to the free-text measurements.
export function formatDimensions(item: PublicItem): string | null {
  const unit = item.measurement_unit === 'cm' ? 'cm' : 'in';
  const parts = [item.length, item.width, item.height].filter(
    (n): n is number => n != null,
  );
  if (parts.length === 0) return item.extra_measurements ?? null;
  const dims = `${parts.join(' × ')} ${unit}`;
  return item.extra_measurements
    ? `${dims} · ${item.extra_measurements}`
    : dims;
}
