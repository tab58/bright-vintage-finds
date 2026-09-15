import { useEffect, useState } from 'react';
import { ChevronLeft, ChevronRight, ImageOff } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  listPublicItemImages,
  type PublicImage,
  type PublicItem,
} from '@/api/public';
import { shop } from './config';
import { formatDimensions, formatPrice } from './format';

// ItemLightbox shows one piece full-size: its photo set plus every public
// detail. The photos are fetched when it opens, so the grid stays one request.
export function ItemLightbox({
  item,
  onClose,
}: {
  item: PublicItem | null;
  onClose: () => void;
}) {
  const [images, setImages] = useState<PublicImage[]>([]);
  const [index, setIndex] = useState(0);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!item) return;
    let cancelled = false;
    setIndex(0);
    setError(null);
    // The cover is already loaded; show it while the full set arrives.
    setImages(
      item.cover_image_url
        ? [{ id: 'cover', url: item.cover_image_url, display_order: 0 }]
        : [],
    );
    listPublicItemImages(item.id)
      .then((rows) => {
        if (!cancelled && rows.length > 0) setImages(rows);
      })
      .catch((e) => {
        if (!cancelled) {
          console.error('loading item photos failed:', e);
          setError('Photos could not be loaded.');
        }
      });
    return () => {
      cancelled = true;
    };
  }, [item]);

  const count = images.length;
  useEffect(() => {
    if (count < 2) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'ArrowRight') setIndex((i) => (i + 1) % count);
      if (e.key === 'ArrowLeft') setIndex((i) => (i - 1 + count) % count);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [count]);

  if (!item) return null;

  const price = formatPrice(item.listing_price_cents);
  const dimensions = formatDimensions(item);
  const current = images[index];

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-h-[92vh] max-w-[min(1000px,calc(100%-2rem))] gap-0 overflow-y-auto border-parlor-rule bg-parlor-paper p-0 sm:max-w-[min(1000px,calc(100%-2rem))]">
        <div className="grid gap-0 md:grid-cols-[3fr_2fr]">
          <div className="relative bg-parlor-paper-dim">
            <div className="aspect-square w-full">
              {current ? (
                <img
                  src={current.url}
                  alt={item.name}
                  className="h-full w-full object-contain"
                />
              ) : (
                <div className="flex h-full w-full items-center justify-center text-parlor-muted">
                  <ImageOff className="size-10" aria-hidden />
                </div>
              )}
            </div>

            {count > 1 && (
              <>
                <PhotoNav
                  side="left"
                  onClick={() => setIndex((i) => (i - 1 + count) % count)}
                />
                <PhotoNav
                  side="right"
                  onClick={() => setIndex((i) => (i + 1) % count)}
                />
                <div className="flex justify-center gap-2 py-3">
                  {images.map((img, i) => (
                    <button
                      key={img.id}
                      type="button"
                      aria-label={`Photo ${i + 1} of ${count}`}
                      aria-current={i === index}
                      onClick={() => setIndex(i)}
                      className={`size-2 rounded-full border border-parlor-wood transition-colors ${
                        i === index ? 'bg-parlor-wood' : 'bg-transparent'
                      }`}
                    />
                  ))}
                </div>
              </>
            )}
          </div>

          <div className="flex flex-col gap-4 p-6 pr-10">
            <div>
              <DialogTitle className="font-display text-2xl leading-tight font-normal text-parlor-ink">
                {item.name}
              </DialogTitle>
              <p className="mt-2 font-display text-lg text-parlor-wood">
                {price ?? 'Ask for price'}
              </p>
            </div>

            {item.description && (
              <DialogDescription className="text-sm leading-relaxed whitespace-pre-line text-parlor-wood">
                {item.description}
              </DialogDescription>
            )}

            <dl className="flex flex-col gap-2 border-t border-parlor-rule pt-4 text-sm">
              <Detail label="Category" value={item.category} />
              <Detail label="Condition" value={item.condition} />
              <Detail label="Measurements" value={dimensions} />
              <Detail
                label="Tags"
                value={item.labels.length > 0 ? item.labels.join(', ') : null}
              />
            </dl>

            {error && <p className="text-xs text-parlor-muted">{error}</p>}

            <a
              href={shop.whatnot.url}
              target="_blank"
              rel="noreferrer"
              className="mt-auto inline-block border border-parlor-brass bg-parlor-brass/10 px-5 py-3 text-center text-[0.7rem] tracking-[0.18em] text-parlor-ink uppercase transition-colors hover:bg-parlor-brass/20"
            >
              Find it on Whatnot
            </a>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function Detail({ label, value }: { label: string; value?: string | null }) {
  if (!value) return null;
  return (
    <div className="flex gap-3">
      <dt className="w-32 shrink-0 text-[0.7rem] tracking-[0.12em] text-parlor-muted uppercase">
        {label}
      </dt>
      <dd className="text-parlor-wood">{value}</dd>
    </div>
  );
}

function PhotoNav({
  side,
  onClick,
}: {
  side: 'left' | 'right';
  onClick: () => void;
}) {
  const Icon = side === 'left' ? ChevronLeft : ChevronRight;
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={side === 'left' ? 'Previous photo' : 'Next photo'}
      className={`absolute top-1/2 ${
        side === 'left' ? 'left-2' : 'right-2'
      } -translate-y-1/2 border border-parlor-rule bg-parlor-paper/85 p-2 text-parlor-wood transition-colors hover:bg-parlor-paper`}
    >
      <Icon className="size-5" aria-hidden />
    </button>
  );
}
