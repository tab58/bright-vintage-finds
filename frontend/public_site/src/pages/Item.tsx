import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  Archive,
  Camera,
  ChevronLeft,
  Images,
  Pencil,
  Plus,
  RotateCcw,
  Tag,
  Trash2,
  Undo2,
} from 'lucide-react'
import { ChipPicker } from '@/components/chip-picker'
import { ACCEPTED_IMAGE_TYPES, toUploadableImage } from '@/components/image-upload'
import {
  emptyFields,
  fieldsFromItem,
  fieldsToBody,
  ItemFieldCards,
  ItemSummaryCards,
  NotesCard,
  type ItemFields,
} from '@/components/item-fields'
import { shortDuration } from '@/components/duration'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  createLabel,
  createSellingPlace,
  deleteItem,
  getItem,
  listItemImages,
  listLabels,
  listSellingPlaces,
  markItemSold,
  updateItem,
  uploadImage,
  type Item,
  type ItemImage,
  type ItemStatus,
  type Label as LabelRow,
  type SellingPlace,
} from '../api/client'

// Native select styled like the shadcn Input; on a phone iOS renders its own
// picker, which beats any custom listbox.
const selectClass =
  'h-11 w-full rounded-md border bg-transparent px-3 text-[15px] shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:opacity-50'

const STATUS_LABEL: Record<ItemStatus, string> = {
  draft: 'Draft',
  listed: 'Active',
  sold: 'Sold',
  archived: 'Archived',
}

const STATUS_STYLE: Record<ItemStatus, string> = {
  draft: 'bg-secondary text-secondary-foreground',
  listed: 'bg-primary text-primary-foreground',
  sold: 'bg-emerald-600 text-white',
  archived: 'bg-muted text-muted-foreground',
}

export default function ItemPage() {
  const { id } = useParams()
  const navigate = useNavigate()

  const [item, setItem] = useState<Item | null>(null)
  const [fields, setFields] = useState<ItemFields>(emptyFields)
  const [images, setImages] = useState<ItemImage[]>([])
  const [places, setPlaces] = useState<SellingPlace[]>([])
  const [labels, setLabels] = useState<LabelRow[]>([])
  const [checkedPlaces, setCheckedPlaces] = useState<Set<string>>(new Set())
  const [checkedLabels, setCheckedLabels] = useState<Set<string>>(new Set())
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  // sale figures, shown read-only on a sold item
  const [soldPrice, setSoldPrice] = useState('')
  const [soldAt, setSoldAt] = useState('')

  const [sellOpen, setSellOpen] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)
  const cameraInput = useRef<HTMLInputElement>(null)
  const libraryInput = useRef<HTMLInputElement>(null)
  // Details are read-only until the owner deliberately opens the form.
  const [editing, setEditing] = useState(false)

  useEffect(() => {
    if (!id) return
    getItem(id)
      .then(seed)
      .catch((e) => setError(String(e)))
    listItemImages(id)
      .then(setImages)
      .catch(() => {})
    listSellingPlaces()
      .then(setPlaces)
      .catch(() => {})
    listLabels()
      .then(setLabels)
      .catch(() => {})
  }, [id])

  // Seed every field from the loaded item; without this a save would blank
  // whatever was left empty.
  function seed(it: Item) {
    setItem(it)
    setFields(fieldsFromItem(it))
    setCheckedPlaces(new Set(it.selling_place_ids))
    setCheckedLabels(new Set(it.label_ids))
    setSoldPrice(it.sold_price_cents != null ? (it.sold_price_cents / 100).toFixed(2) : '')
    setSoldAt(it.sold_at?.slice(0, 10) ?? '')
  }

  async function run(what: string, fn: () => Promise<Item>) {
    setBusy(true)
    setError(null)
    try {
      seed(await fn())
    } catch (e) {
      setError(`${what} failed: ${String(e)}`)
    } finally {
      setBusy(false)
    }
  }

  // Photos upload straight away here: the item already exists, so there is
  // nothing to stage them against.
  async function addPhotos(fileList: FileList | null, itemId: string) {
    const picked = Array.from(fileList ?? [])
    if (picked.length === 0) return
    setBusy(true)
    setError(null)
    try {
      for (const file of picked) await uploadImage(itemId, await toUploadableImage(file))
      setImages(await listItemImages(itemId))
    } catch (e) {
      setError(`Photo upload failed: ${String(e instanceof Error ? e.message : e)}`)
    } finally {
      setBusy(false)
    }
  }

  if (error && !item) {
    return (
      <main className="mx-auto flex max-w-[480px] flex-col gap-2 p-4 pt-[calc(1rem+env(safe-area-inset-top,0px))]">
        <p className="text-sm text-destructive">{error}</p>
        <Link to="/inventory" className="text-sm underline">
          ← Back
        </Link>
      </main>
    )
  }
  if (!item) {
    return (
      <main className="mx-auto max-w-[480px] p-4 pt-[calc(1rem+env(safe-area-inset-top,0px))]">
        <p className="text-sm text-muted-foreground">Loading…</p>
      </main>
    )
  }

  // Sold and archived items are records, not forms: nothing on them is editable.
  const readOnly = item.status === 'sold' || item.status === 'archived'
  // Photos stay editable while the item is still a draft; once listed it is
  // out in the world and the pictures are part of the listing.
  const canAddPhotos = item.status === 'draft'
  // Never listed: the item never left the workbench, so it can just go away.
  // Anything that reached listed is archived instead, and the API enforces it.
  const canDelete = item.status === 'draft' && !item.first_listed_at
  const canList = checkedPlaces.size > 0

  function toggle(set: Set<string>, id: string, apply: (s: Set<string>) => void) {
    const next = new Set(set)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    apply(next)
  }

  const saveEdits = () =>
    run('Save', () =>
      updateItem(item.id, {
        ...fieldsToBody(fields),
        selling_place_ids: [...checkedPlaces],
        label_ids: [...checkedLabels],
      }),
    )

  const setStatus = (status: ItemStatus, what: string) =>
    run(what, () => updateItem(item.id, { name: item.name, status }))

  async function addPlace(name: string) {
    try {
      const p = await createSellingPlace(name)
      setPlaces((prev) => [...prev, p])
      setCheckedPlaces((prev) => new Set(prev).add(p.id))
    } catch (e) {
      setError(String(e))
    }
  }

  async function addLabel(name: string) {
    try {
      const l = await createLabel(name)
      setLabels((prev) => [...prev, l])
      setCheckedLabels((prev) => new Set(prev).add(l.id))
    } catch (e) {
      setError(String(e))
    }
  }

  const subtitle =
    item.status === 'listed' && item.listed_at
      ? `Listed ${shortDuration(item.listed_at)} ago`
      : item.status === 'sold' && item.sold_at
        ? `Sold ${item.sold_at.slice(0, 10)}${item.first_listed_at ? ` · ${shortDuration(item.first_listed_at, item.sold_at)} to sell` : ''}`
        : item.whatnot_number
          ? `WN ${item.whatnot_number}`
          : 'No Whatnot number'

  return (
    <div className="min-h-dvh bg-muted/40">
      <header className="sticky top-0 z-30 border-b bg-background/85 pt-[env(safe-area-inset-top,0px)] backdrop-blur-md">
        <div className="flex h-14 items-center gap-1 px-2">
          <Button variant="ghost" size="icon" onClick={() => navigate('/inventory')} aria-label="Back">
            <ChevronLeft className="size-5" />
          </Button>
          <div className="min-w-0 flex-1">
            <h1 className="truncate text-[17px] leading-tight font-semibold tracking-tight">{item.name}</h1>
            <p className="truncate text-xs text-muted-foreground">{subtitle}</p>
          </div>
          <span className={`rounded-full px-2.5 py-1 text-[11px] font-semibold ${STATUS_STYLE[item.status]}`}>
            {STATUS_LABEL[item.status]}
          </span>
        </div>
      </header>

      {images.length > 0 && (
        <div className="border-b bg-background">
          <div className="flex gap-2 overflow-x-auto px-4 py-3">
            {images.map((img, i) => (
              <div key={img.id} className="relative size-24 shrink-0 overflow-hidden rounded-md border">
                <img src={img.url} alt={item.name} className="size-full object-cover" />
                {i === 0 && (
                  <span className="absolute top-1 left-1 rounded bg-black/60 px-1.5 py-0.5 text-[10px] font-medium text-white">
                    Cover
                  </span>
                )}
              </div>
            ))}
            {canAddPhotos && (
              <button
                type="button"
                onClick={() => libraryInput.current?.click()}
                aria-label="Add photos"
                className="flex size-24 shrink-0 items-center justify-center rounded-md border border-dashed bg-muted/50 text-muted-foreground"
              >
                <Plus className="size-5" />
              </button>
            )}
          </div>
        </div>
      )}

      <main className="space-y-4 px-4 pt-4 pb-44">
        {readOnly && (
          <p className="rounded-md border bg-background px-3 py-2 text-sm text-muted-foreground">
            {item.status === 'sold'
              ? 'Sold items are a record of the sale and cannot be edited.'
              : 'Archived items are read-only. Restore it to draft to make changes.'}
          </p>
        )}

        {item.status === 'sold' && (
          <Card>
            <CardContent className="space-y-0">
              <h2 className="pb-2 text-sm font-semibold">Sale</h2>
              <SaleRow label="Sold for" value={soldPrice ? `$${soldPrice}` : '—'} />
              <SaleRow label="Sold on" value={soldAt || '—'} />
              <SaleRow label="Sold at" value={item.sold_place ?? '—'} />
              <SaleRow label="Whatnot number" value={fields.whatnot || '—'} />
            </CardContent>
          </Card>
        )}

        {canAddPhotos && (
          <Card>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between">
                <h2 className="text-sm font-semibold">Photos</h2>
                <span className="rounded-full bg-secondary px-2 py-0.5 text-[11px] font-medium text-secondary-foreground">
                  {images.length}
                </span>
              </div>
              <div className="flex gap-2">
                <Button className="h-11 flex-1" disabled={busy} onClick={() => cameraInput.current?.click()}>
                  <Camera /> {busy ? 'Uploading…' : 'Take photo'}
                </Button>
                <Button
                  variant="outline"
                  className="h-11 flex-1"
                  disabled={busy}
                  onClick={() => libraryInput.current?.click()}
                >
                  <Images /> Library
                </Button>
              </div>
              <input
                ref={cameraInput}
                type="file"
                accept={ACCEPTED_IMAGE_TYPES}
                capture="environment"
                hidden
                onChange={(e) => {
                  void addPhotos(e.target.files, item.id)
                  e.target.value = ''
                }}
              />
              <input
                ref={libraryInput}
                type="file"
                accept={ACCEPTED_IMAGE_TYPES}
                multiple
                hidden
                onChange={(e) => {
                  void addPhotos(e.target.files, item.id)
                  e.target.value = ''
                }}
              />
            </CardContent>
          </Card>
        )}

        {editing ? (
          <>
            <ItemFieldCards value={fields} onChange={setFields} idPrefix="item" />
            <div className="flex gap-2">
              <Button
                variant="ghost"
                className="h-11 flex-1"
                disabled={busy}
                onClick={() => {
                  setFields(fieldsFromItem(item))
                  setEditing(false)
                }}
              >
                Cancel
              </Button>
              <Button
                className="h-11 flex-1"
                disabled={busy}
                onClick={async () => {
                  await saveEdits()
                  setEditing(false)
                }}
              >
                Save details
              </Button>
            </div>
          </>
        ) : (
          <>
            <ItemSummaryCards value={fields} />
            {!readOnly && (
              <Button variant="outline" className="h-11 w-full" onClick={() => setEditing(true)}>
                <Pencil /> Edit details
              </Button>
            )}
          </>
        )}

        {!editing && !readOnly && (
          <Card>
            <CardContent className="space-y-2">
              <Label htmlFor="item-whatnot">Whatnot number</Label>
              <Input
                id="item-whatnot"
                className="h-11"
                inputMode="numeric"
                placeholder="e.g. 214"
                value={fields.whatnot}
                onChange={(e) => setFields({ ...fields, whatnot: e.target.value })}
              />
            </CardContent>
          </Card>
        )}

        {!editing && (
          <ChipPicker
            title="Selling places"
            hint={item.status === 'draft' ? 'Pick at least one to list it' : 'Where this item is listed'}
            rows={places}
            checked={checkedPlaces}
            onToggle={(id) => toggle(checkedPlaces, id, setCheckedPlaces)}
            onAdd={addPlace}
            addPlaceholder="Add place…"
            disabled={readOnly}
          />
        )}

        {!editing && (
          <ChipPicker
            title="Labels"
            hint="For searching later"
            rows={labels}
            checked={checkedLabels}
            onToggle={(id) => toggle(checkedLabels, id, setCheckedLabels)}
            onAdd={addLabel}
            addPlaceholder="Add label…"
            disabled={readOnly}
          />
        )}

        {!editing &&
          (readOnly ? (
            <Card>
              <CardContent className="space-y-1">
                <h2 className="text-sm font-semibold">Notes</h2>
                <p className="text-[15px] whitespace-pre-wrap text-muted-foreground">{fields.notes || '—'}</p>
              </CardContent>
            </Card>
          ) : (
            <NotesCard value={fields} onChange={setFields} idPrefix="item" />
          ))}

        {!readOnly && !editing && (
          <Button variant="outline" className="h-11 w-full" onClick={saveEdits} disabled={busy}>
            Save changes
          </Button>
        )}

        {error && (
          <p className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {error}
          </p>
        )}
      </main>

      <div className="fixed inset-x-0 bottom-0 z-30 border-t bg-background/90 pb-[calc(1.5rem+env(safe-area-inset-bottom,0px))] backdrop-blur-md">
        <div className="flex flex-col gap-2 px-4 pt-3">
          {item.status === 'draft' && (
            <>
              <Button
                className="h-12 w-full text-[15px]"
                disabled={busy || !canList}
                onClick={() => setStatus('listed', 'List')}
              >
                <Tag /> List it
              </Button>
              {!canList && (
                <p className="text-center text-xs text-muted-foreground">Pick a selling place first</p>
              )}
              {canDelete ? (
                <Button
                  variant="ghost"
                  className="h-9 w-full text-destructive hover:text-destructive"
                  disabled={busy}
                  onClick={() => setConfirmDelete(true)}
                >
                  <Trash2 /> Delete item
                </Button>
              ) : (
                <Button
                  variant="ghost"
                  className="h-9 w-full text-muted-foreground"
                  disabled={busy}
                  onClick={() => setStatus('archived', 'Archive')}
                >
                  <Archive /> Archive
                </Button>
              )}
            </>
          )}

          {item.status === 'listed' && (
            <>
              <Button className="h-12 w-full text-[15px]" disabled={busy} onClick={() => setSellOpen(true)}>
                Mark sold
              </Button>
              <div className="flex gap-2">
                <Button
                  variant="ghost"
                  className="h-9 flex-1 text-muted-foreground"
                  disabled={busy}
                  onClick={() => setStatus('draft', 'Unlist')}
                >
                  <Undo2 /> Unlist
                </Button>
                <Button
                  variant="ghost"
                  className="h-9 flex-1 text-muted-foreground"
                  disabled={busy}
                  onClick={() => setStatus('archived', 'Archive')}
                >
                  <Archive /> Archive
                </Button>
              </div>
            </>
          )}

          {item.status === 'archived' && (
            <Button
              className="h-12 w-full text-[15px]"
              disabled={busy}
              onClick={() => setStatus('draft', 'Restore')}
            >
              <RotateCcw /> Restore to draft
            </Button>
          )}

          {item.status === 'sold' && (
            <p className="text-center text-sm text-muted-foreground">
              Sold {item.sold_at?.slice(0, 10)}
              {item.sold_place ? ` · ${item.sold_place}` : ''}
            </p>
          )}
        </div>
      </div>

      <Dialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <DialogContent className="sm:max-w-[360px]">
          <DialogHeader>
            <DialogTitle>Delete this item?</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            “{item.name}” and its photos will be removed. This cannot be undone from the app.
          </p>
          <DialogFooter>
            <div className="flex w-full gap-2">
              <Button variant="ghost" className="h-11 flex-1" onClick={() => setConfirmDelete(false)}>
                Cancel
              </Button>
              <Button
                className="h-11 flex-1 bg-destructive text-white hover:bg-destructive/90"
                disabled={busy}
                onClick={async () => {
                  setBusy(true)
                  try {
                    await deleteItem(item.id)
                    navigate('/inventory')
                  } catch (e) {
                    setError(`Delete failed: ${String(e)}`)
                    setBusy(false)
                    setConfirmDelete(false)
                  }
                }}
              >
                Delete
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <MarkSoldDialog
        open={sellOpen}
        onOpenChange={setSellOpen}
        places={places}
        defaultPlaceId={item.selling_place_ids[0] ?? ''}
        onCreatePlace={async (name) => {
          const p = await createSellingPlace(name)
          setPlaces((prev) => [...prev, p])
          return p
        }}
        onConfirm={async (sale) => {
          await run('Mark sold', () => markItemSold(item.id, sale))
          setSellOpen(false)
        }}
      />
    </div>
  )
}

// Sentinel for the "Custom…" option: picking it reveals a name field, and
// confirming creates a real selling place so the sale keeps its FK.
const CUSTOM_PLACE = '__custom__'

function SaleRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-4 border-b py-2 last:border-b-0">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-right text-[15px]">{value}</span>
    </div>
  )
}

function MarkSoldDialog({
  open,
  onOpenChange,
  places,
  defaultPlaceId,
  onCreatePlace,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  places: SellingPlace[]
  defaultPlaceId: string
  onCreatePlace: (name: string) => Promise<SellingPlace>
  onConfirm: (sale: { sold_at: string; sold_price_cents: number; sold_place_id: string }) => Promise<void>
}) {
  const [price, setPrice] = useState('')
  const [date, setDate] = useState(new Date().toISOString().slice(0, 10))
  const [placeId, setPlaceId] = useState(defaultPlaceId)
  const [customName, setCustomName] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open) {
      setPlaceId(defaultPlaceId)
      setCustomName('')
      setError(null)
    }
  }, [open, defaultPlaceId])

  const custom = placeId === CUSTOM_PLACE
  const ready = price !== '' && (custom ? customName.trim() !== '' : placeId !== '')

  // Re-use a place that already exists under that name rather than tripping
  // the unique-name constraint with a near-duplicate.
  async function resolvePlaceId(): Promise<string> {
    if (!custom) return placeId
    const name = customName.trim()
    const existing = places.find((p) => p.name.toLowerCase() === name.toLowerCase())
    if (existing) return existing.id
    const created = await onCreatePlace(name)
    return created.id
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[420px]">
        <DialogHeader>
          <DialogTitle>Mark sold</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="sale-place">Sold at</Label>
            <select
              id="sale-place"
              className={selectClass}
              value={placeId}
              onChange={(e) => setPlaceId(e.target.value)}
            >
              <option value="">Choose a place…</option>
              {places.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
              <option value={CUSTOM_PLACE}>Custom…</option>
            </select>
          </div>
          {custom && (
            <div className="space-y-2">
              <Label htmlFor="sale-custom">New place name</Label>
              <Input
                id="sale-custom"
                className="h-11"
                placeholder="Pickup, flea market, a friend…"
                value={customName}
                onChange={(e) => setCustomName(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                Saved to your selling places, so you can filter by it later.
              </p>
            </div>
          )}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor="sale-price">Sale price</Label>
              <div className="relative">
                <span className="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground">
                  $
                </span>
                <Input
                  id="sale-price"
                  className="h-11 pl-7"
                  inputMode="decimal"
                  placeholder="0.00"
                  value={price}
                  onChange={(e) => setPrice(e.target.value)}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="sale-date">Date</Label>
              <Input
                id="sale-date"
                type="date"
                className="h-11"
                value={date}
                onChange={(e) => setDate(e.target.value)}
              />
            </div>
          </div>
        </div>
        {error && <p className="text-sm text-destructive">{error}</p>}
        <DialogFooter>
          <Button
            className="h-11 w-full"
            disabled={!ready || saving}
            onClick={async () => {
              setSaving(true)
              try {
                await onConfirm({
                  sold_at: new Date(date).toISOString(),
                  sold_price_cents: Math.round(parseFloat(price) * 100),
                  sold_place_id: await resolvePlaceId(),
                })
              } catch (e) {
                setError(String(e))
              } finally {
                setSaving(false)
              }
            }}
          >
            {saving ? 'Saving…' : 'Confirm sale'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
