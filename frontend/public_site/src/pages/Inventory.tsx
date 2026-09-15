import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ChevronDown, ChevronRight, ImageOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { shortDuration } from '@/components/duration'
import { Input } from '@/components/ui/input'
import {
  listItems,
  listLabels,
  listSellingPlaces,
  deleteItem,
  updateItem,
  type Item,
  type ItemStatus,
  type Label,
  type SellingPlace,
} from '../api/client'

// Status groups, in the order items move through them. `listed` shows as
// "Active"; the API value stays `listed`. Finished groups start collapsed so
// old stock doesn't bury today's work.
const GROUPS: { value: ItemStatus; label: string; openByDefault: boolean }[] = [
  { value: 'draft', label: 'Draft', openByDefault: true },
  { value: 'listed', label: 'Active', openByDefault: true },
  { value: 'sold', label: 'Sold', openByDefault: false },
  { value: 'archived', label: 'Archived', openByDefault: false },
]

// Native selects styled like the shadcn Input; no Select component installed yet.
const selectClass =
  'h-9 rounded-md border bg-transparent px-3 py-1 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50'

const LONG_PRESS_MS = 500

export default function InventoryPage() {
  const navigate = useNavigate()
  const [items, setItems] = useState<Item[]>([])
  const [places, setPlaces] = useState<SellingPlace[]>([])
  const [labels, setLabels] = useState<Label[]>([])
  const [query, setQuery] = useState('')
  const [placeId, setPlaceId] = useState('')
  const [labelId, setLabelId] = useState('')
  const [whatnot, setWhatnot] = useState('')
  const [status, setStatus] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [open, setOpen] = useState<Set<ItemStatus>>(
    () => new Set(GROUPS.filter((g) => g.openByDefault).map((g) => g.value)),
  )
  const [menuItem, setMenuItem] = useState<Item | null>(null)
  const [reloadKey, setReloadKey] = useState(0)

  useEffect(() => {
    listSellingPlaces().then(setPlaces).catch((e) => setError(String(e)))
    listLabels().then(setLabels).catch(() => {})
  }, [])

  useEffect(() => {
    let cancelled = false
    listItems({ query, place_id: placeId, label_id: labelId, whatnot_number: whatnot, status })
      .then((rows) => {
        if (!cancelled) {
          setItems(rows)
          setError(null)
        }
      })
      .catch((e) => setError(String(e)))
    return () => {
      cancelled = true
    }
  }, [query, placeId, labelId, whatnot, status, reloadKey])

  function toggleGroup(value: ItemStatus) {
    const next = new Set(open)
    if (next.has(value)) next.delete(value)
    else next.add(value)
    setOpen(next)
  }

  async function remove(it: Item) {
    setMenuItem(null)
    try {
      await deleteItem(it.id)
      setReloadKey((k) => k + 1)
    } catch (e) {
      setError(String(e))
    }
  }

  async function move(it: Item, to: ItemStatus) {
    setMenuItem(null)
    try {
      await updateItem(it.id, { name: it.name, status: to })
      setReloadKey((k) => k + 1)
    } catch (e) {
      setError(String(e))
    }
  }

  return (
    <main className="mx-auto max-w-[640px] p-4 pt-[calc(1rem+env(safe-area-inset-top,0px))]">
      <div className="mb-3 flex items-center justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">Inventory</h1>
        <Button onClick={() => navigate('/inventory/new')}>New item</Button>
      </div>
      <div className="mb-3 flex flex-wrap gap-2">
        <Input
          className="w-auto flex-1"
          placeholder="Search name…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <select className={selectClass} value={placeId} onChange={(e) => setPlaceId(e.target.value)}>
          <option value="">Any place</option>
          {places.map((p) => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
        <select className={selectClass} value={labelId} onChange={(e) => setLabelId(e.target.value)}>
          <option value="">Any label</option>
          {labels.map((l) => (
            <option key={l.id} value={l.id}>{l.name}</option>
          ))}
        </select>
        <Input
          className="w-28"
          placeholder="Whatnot #"
          value={whatnot}
          onChange={(e) => setWhatnot(e.target.value)}
        />
        <select className={selectClass} value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">Any status</option>
          {GROUPS.map((g) => (
            <option key={g.value} value={g.value}>{g.label}</option>
          ))}
        </select>
      </div>
      {error && <p className="text-sm text-destructive">{error}</p>}

      {GROUPS.map(({ value, label }) => {
        // Newest first: KSUID ids and created_at both sort by creation time.
        const rows = items
          .filter((it) => it.status === value)
          .sort((a, b) => b.created_at.localeCompare(a.created_at))
        if (rows.length === 0) return null
        const isOpen = open.has(value)
        return (
          <section key={value} className="mb-4">
            <button
              type="button"
              onClick={() => toggleGroup(value)}
              className="sticky top-0 flex w-full items-center gap-1.5 border-b bg-background py-2 text-sm font-semibold"
            >
              {isOpen ? <ChevronDown className="size-4" /> : <ChevronRight className="size-4" />}
              {label}
              <span className="text-xs font-normal text-muted-foreground">{rows.length}</span>
            </button>
            {isOpen && (
              <ul className="list-none p-0">
                {rows.map((it) => (
                  <ItemRow key={it.id} item={it} onLongPress={() => setMenuItem(it)} />
                ))}
              </ul>
            )}
          </section>
        )
      })}
      {items.length === 0 && !error && <p className="text-sm text-muted-foreground">No items match.</p>}

      <ItemActions
        item={menuItem}
        onClose={() => setMenuItem(null)}
        onMove={move}
        onDelete={remove}
      />
    </main>
  )
}

// Right-hand summary: listing age, Whatnot number, sale price — whichever apply.
function meta(item: Item): string[] {
  const parts: string[] = []
  // Active: how long it has been listed. Sold: how long it took to sell,
  // measured from the first listing.
  if (item.status === 'listed' && item.listed_at) {
    parts.push(shortDuration(item.listed_at))
  }
  if (item.status === 'sold' && item.first_listed_at && item.sold_at) {
    parts.push(`sold in ${shortDuration(item.first_listed_at, item.sold_at)}`)
  }
  if (item.whatnot_number) parts.push(`WN ${item.whatnot_number}`)
  if (item.sold_price_cents != null) parts.push(`$${(item.sold_price_cents / 100).toFixed(2)}`)
  return parts
}

function ItemRow({ item, onLongPress }: { item: Item; onLongPress: () => void }) {
  const timer = useRef<number | null>(null)
  const fired = useRef(false)

  function start() {
    fired.current = false
    timer.current = window.setTimeout(() => {
      fired.current = true
      onLongPress()
    }, LONG_PRESS_MS)
  }

  function cancel() {
    if (timer.current !== null) window.clearTimeout(timer.current)
    timer.current = null
  }

  return (
    <li className="border-b">
      <Link
        to={`/inventory/item/${item.id}`}
        // A long press opens the action sheet instead of navigating.
        onClick={(e) => {
          if (fired.current) e.preventDefault()
        }}
        onContextMenu={(e) => {
          e.preventDefault()
          onLongPress()
        }}
        onPointerDown={start}
        onPointerUp={cancel}
        onPointerLeave={cancel}
        onPointerCancel={cancel}
        className="flex items-center gap-3 px-1 py-2 text-foreground no-underline select-none"
      >
        {item.cover_image_url ? (
          <img
            src={item.cover_image_url}
            alt=""
            className="size-12 shrink-0 rounded-md border object-cover"
          />
        ) : (
          // Keep the row aligned whether or not the item has photos yet.
          <span className="flex size-12 shrink-0 items-center justify-center rounded-md border border-dashed bg-muted/50 text-muted-foreground">
            <ImageOff className="size-4" />
          </span>
        )}
        <span className="min-w-0 flex-1 truncate font-semibold">{item.name}</span>
        <span className="shrink-0 text-[13px] text-muted-foreground">{meta(item).join(' · ')}</span>
      </Link>
    </li>
  )
}

// The moves each status allows. Selling needs a place and a price, so it lives
// on the item page rather than here. Deleting a draft is handled separately:
// it is not a status change.
const MOVES: Record<ItemStatus, { to: ItemStatus; label: string }[]> = {
  draft: [
    { to: 'listed', label: 'List it' },
    { to: 'archived', label: 'Archive' },
  ],
  listed: [
    { to: 'draft', label: 'Unlist' },
    { to: 'archived', label: 'Archive' },
  ],
  sold: [],
  archived: [{ to: 'draft', label: 'Restore to draft' }],
}

function ItemActions({
  item,
  onClose,
  onMove,
  onDelete,
}: {
  item: Item | null
  onClose: () => void
  onMove: (item: Item, to: ItemStatus) => void
  onDelete: (item: Item) => void
}) {
  if (!item) return null
  const neverListed = item.status === 'draft' && !item.first_listed_at
  const moves = MOVES[item.status].filter((m) => !(neverListed && m.to === 'archived'))
  const canList = item.selling_place_ids.length > 0

  return (
    <Dialog open onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="sm:max-w-[360px]">
        <DialogHeader>
          <DialogTitle className="truncate">{item.name}</DialogTitle>
        </DialogHeader>
        <div className="flex flex-col gap-2">
          {moves.map((m) => {
            const blocked = m.to === 'listed' && !canList
            return (
              <Button
                key={m.to}
                variant="outline"
                className="h-11 w-full"
                disabled={blocked}
                onClick={() => onMove(item, m.to)}
              >
                {m.label}
              </Button>
            )
          })}
          {neverListed && (
            <Button
              variant="outline"
              className="h-11 w-full text-destructive hover:text-destructive"
              onClick={() => onDelete(item)}
            >
              Delete
            </Button>
          )}
          {moves.length === 0 && (
            <p className="text-sm text-muted-foreground">Sold items stay put. Open it to fix the sale.</p>
          )}
          {moves.some((m) => m.to === 'listed') && !canList && (
            <p className="text-xs text-muted-foreground">
              Needs a selling place — open the item to pick one.
            </p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
