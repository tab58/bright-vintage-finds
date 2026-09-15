import { ImageOff } from 'lucide-react';
import type { PublicItem } from '@/api/public';
import { formatPrice } from './format';

// ItemCard is one piece in the catalog grid: cover photo, name, price and a
// short caption. Clicking it opens the lightbox.
export function ItemCard({
  item,
  onOpen,
}: {
  item: PublicItem;
  onOpen: (item: PublicItem) => void;
}) {
  const price = formatPrice(item.listing_price_cents);
  const caption = [item.category, item.condition].filter(Boolean).join(' · ');

  return (
    <button
      type="button"
      onClick={() => onOpen(item)}
      className="group flex flex-col border border-parlor-rule bg-parlor-paper text-left shadow-[0_1px_0_#fff_inset,0_6px_18px_-12px_rgba(46,36,24,0.45)] transition-shadow hover:shadow-[0_1px_0_#fff_inset,0_14px_30px_-14px_rgba(46,36,24,0.55)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-parlor-brass"
    >
      <div className="aspect-square overflow-hidden border-b border-parlor-rule bg-parlor-paper-dim">
        {item.cover_image_url ? (
          <img
            src={item.cover_image_url}
            alt={item.name}
            loading="lazy"
            className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.04]"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-parlor-muted">
            <ImageOff className="size-8" aria-hidden />
          </div>
        )}
      </div>

      <div className="flex grow flex-col gap-1 px-4 py-3">
        <h3 className="font-display text-base leading-snug text-parlor-ink">
          {item.name}
        </h3>
        {caption && (
          <p className="line-clamp-2 text-[0.7rem] tracking-[0.12em] text-parlor-muted uppercase">
            {caption}
          </p>
        )}
        <p className="mt-auto pt-2 font-display text-sm text-parlor-wood">
          {price ?? 'Ask for price'}
        </p>
      </div>
    </button>
  );
}
