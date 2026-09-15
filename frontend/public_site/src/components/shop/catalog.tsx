import { useEffect, useState } from 'react';
import { Search } from 'lucide-react';
import {
  getPublicFilters,
  listPublicItems,
  type PublicItem,
} from '@/api/public';
import { ItemCard } from './item-card';
import { ItemLightbox } from './item-lightbox';

const PAGE_SIZE = 24;
const SEARCH_DEBOUNCE_MS = 300;

// Native selects styled to match the shop, as the inventory pages do: no
// Select primitive is installed.
const selectClass =
  'h-10 border border-parlor-rule bg-parlor-paper px-3 font-display text-sm text-parlor-ink outline-none focus-visible:border-parlor-brass';

// Catalog is the shop grid: server-side search and filters, cursor paging, and
// a lightbox for the selected piece.
export function Catalog() {
  const [query, setQuery] = useState('');
  const [search, setSearch] = useState('');
  const [category, setCategory] = useState('');
  const [label, setLabel] = useState('');
  const [categories, setCategories] = useState<string[]>([]);
  const [labels, setLabels] = useState<string[]>([]);
  const [items, setItems] = useState<PublicItem[]>([]);
  const [cursor, setCursor] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<PublicItem | null>(null);

  useEffect(() => {
    getPublicFilters()
      .then((filters) => {
        setCategories(filters.categories);
        setLabels(filters.labels);
      })
      .catch((e) => console.error('loading catalog filters failed:', e));
  }, []);

  // Typing shouldn't fire a request per keystroke.
  useEffect(() => {
    const timer = setTimeout(() => setSearch(query.trim()), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [query]);

  // Any filter change restarts paging from the first page.
  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    listPublicItems({ query: search, category, label, limit: PAGE_SIZE })
      .then((page) => {
        if (cancelled) return;
        setItems(page.items);
        setCursor(page.next_cursor);
        setError(null);
      })
      .catch((e) => {
        if (cancelled) return;
        console.error('loading the catalog failed:', e);
        setError(
          'The collection could not be loaded. Please try again shortly.',
        );
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [search, category, label]);

  async function loadMore() {
    if (!cursor || loadingMore) return;
    setLoadingMore(true);
    try {
      const page = await listPublicItems({
        query: search,
        category,
        label,
        cursor,
        limit: PAGE_SIZE,
      });
      setItems((current) => [...current, ...page.items]);
      setCursor(page.next_cursor);
      setError(null);
    } catch (e) {
      console.error('loading more items failed:', e);
      setError('Could not load more pieces. Please try again shortly.');
    } finally {
      setLoadingMore(false);
    }
  }

  return (
    <section id="collection" className="mx-auto w-full max-w-6xl px-6 py-16">
      <header className="flex flex-col items-center gap-3 text-center">
        <h2 className="font-display text-3xl text-parlor-ink">In the shop</h2>
        <p className="max-w-xl text-sm leading-relaxed text-parlor-muted">
          Every piece is one of one. When it sells, it leaves the shelf.
        </p>
      </header>

      <div className="mt-10 flex flex-col gap-3 border-y border-parlor-rule py-4 sm:flex-row sm:items-center">
        <div className="relative grow">
          <Search
            className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-parlor-muted"
            aria-hidden
          />
          <input
            type="search"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search the collection"
            aria-label="Search the collection"
            className="h-10 w-full border border-parlor-rule bg-parlor-paper pr-3 pl-9 font-display text-sm text-parlor-ink outline-none placeholder:text-parlor-muted focus-visible:border-parlor-brass"
          />
        </div>

        {categories.length > 0 && (
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            aria-label="Filter by category"
            className={selectClass}
          >
            <option value="">All categories</option>
            {categories.map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        )}

        {labels.length > 0 && (
          <select
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            aria-label="Filter by tag"
            className={selectClass}
          >
            <option value="">All tags</option>
            {labels.map((l) => (
              <option key={l} value={l}>
                {l}
              </option>
            ))}
          </select>
        )}
      </div>

      {error && (
        <p className="mt-8 text-center text-sm text-parlor-wood" role="alert">
          {error}
        </p>
      )}

      {loading ? (
        <p className="mt-16 text-center text-sm tracking-[0.14em] text-parlor-muted uppercase">
          Opening the cabinet…
        </p>
      ) : items.length === 0 ? (
        <p className="mt-16 text-center text-sm leading-relaxed text-parlor-muted">
          Nothing matches that just now. Try a different search, or follow along
          for the next batch.
        </p>
      ) : (
        <div className="mt-8 grid grid-cols-2 gap-5 sm:grid-cols-3 lg:grid-cols-4">
          {items.map((item) => (
            <ItemCard key={item.id} item={item} onOpen={setSelected} />
          ))}
        </div>
      )}

      {cursor && !loading && (
        <div className="mt-12 flex justify-center">
          <button
            type="button"
            onClick={loadMore}
            disabled={loadingMore}
            className="border border-parlor-wood px-8 py-3 text-[0.7rem] tracking-[0.18em] text-parlor-wood uppercase transition-colors hover:bg-parlor-wood hover:text-parlor-paper disabled:opacity-60"
          >
            {loadingMore ? 'Loading…' : 'Show more'}
          </button>
        </div>
      )}

      <ItemLightbox item={selected} onClose={() => setSelected(null)} />
    </section>
  );
}
