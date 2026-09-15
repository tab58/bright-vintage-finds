import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Camera, ChevronLeft, Images, Plus, X } from 'lucide-react';
import { ChipPicker } from '@/components/chip-picker';
import {
  ACCEPTED_IMAGE_TYPES,
  toUploadableImage,
} from '@/components/image-upload';
import {
  emptyFields,
  fieldsToBody,
  ItemFieldCards,
  NotesCard,
  type ItemFields,
} from '@/components/item-fields';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  createItem,
  createLabel,
  createSellingPlace,
  listLabels,
  listSellingPlaces,
  uploadImage,
  type ItemImage,
  type Label as LabelRow,
  type SellingPlace,
} from '../api/client';

export default function IntakePage() {
  const navigate = useNavigate();

  const [fields, setFields] = useState<ItemFields>(emptyFields);

  const [places, setPlaces] = useState<SellingPlace[]>([]);
  const [checkedPlaces, setCheckedPlaces] = useState<Set<string>>(new Set());
  const [labels, setLabels] = useState<LabelRow[]>([]);
  const [checkedLabels, setCheckedLabels] = useState<Set<string>>(new Set());

  // photos staged before save
  const [photos, setPhotos] = useState<File[]>([]);
  const cameraInput = useRef<HTMLInputElement>(null);
  const libraryInput = useRef<HTMLInputElement>(null);
  const [saving, setSaving] = useState(false);
  const [converting, setConverting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Batch intake: saving clears the form and stays here, so a pile of items
  // goes in one after another. The banner is the receipt for the last one.
  const [lastSaved, setLastSaved] = useState<{
    id: string;
    name: string;
  } | null>(null);
  const [batchCount, setBatchCount] = useState(0);

  useEffect(() => {
    listSellingPlaces()
      .then(setPlaces)
      .catch(() => {});
    listLabels()
      .then(setLabels)
      .catch(() => {});
  }, []);

  const previews = useMemo(
    () => photos.map((f) => URL.createObjectURL(f)),
    [photos],
  );
  useEffect(
    () => () => previews.forEach((url) => URL.revokeObjectURL(url)),
    [previews],
  );

  function toggle(
    set: Set<string>,
    id: string,
    apply: (s: Set<string>) => void,
  ) {
    const next = new Set(set);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    apply(next);
  }

  async function addPhotos(fileList: FileList | null) {
    const picked = Array.from(fileList ?? []);
    if (picked.length === 0) return;
    setConverting(true);
    try {
      // Convert here, not at save time: a photo that cannot be used should
      // fail while the item is still on screen, not after it is saved.
      const usable = await Promise.all(picked.map(toUploadableImage));
      setPhotos((prev) => [...prev, ...usable]);
    } catch (e) {
      setError(String(e instanceof Error ? e.message : e));
    } finally {
      setConverting(false);
    }
  }

  function removePhoto(index: number) {
    setPhotos((prev) => prev.filter((_, i) => i !== index));
  }

  async function addPlace(name: string) {
    try {
      const p = await createSellingPlace(name);
      setPlaces((prev) => [...prev, p]);
      setCheckedPlaces((prev) => new Set(prev).add(p.id));
    } catch (e) {
      setError(String(e));
    }
  }

  async function addLabel(name: string) {
    try {
      const l = await createLabel(name);
      setLabels((prev) => [...prev, l]);
      setCheckedLabels((prev) => new Set(prev).add(l.id));
    } catch (e) {
      setError(String(e));
    }
  }

  async function save() {
    if (!fields.name.trim()) {
      setError('Name is required');
      return;
    }
    setSaving(true);
    setError(null);
    try {
      const item = await createItem({
        ...fieldsToBody(fields),
        selling_place_ids: [...checkedPlaces],
        label_ids: [...checkedLabels],
      });
      // Upload staged photos sequentially; failures surface but keep saved item.
      const uploaded: ItemImage[] = [];
      for (const file of photos) {
        try {
          uploaded.push(await uploadImage(item.id, file));
        } catch (e) {
          const msg = String(e).includes('422')
            ? `Item saved, but "${file.name}" is not a JPEG, PNG or WebP — iPhone HEIC photos need converting first.`
            : `Item saved, but a photo failed to upload: ${String(e)}`;
          setError(msg);
        }
      }
      void uploaded;
      // Keep the selling places and labels ticked: a batch usually shares them.
      setFields(emptyFields);
      setPhotos([]);
      setLastSaved({ id: item.id, name: item.name });
      setBatchCount((n) => n + 1);
      setSaving(false);
      window.scrollTo({ top: 0 });
      document.getElementById('new-name')?.focus();
    } catch (e) {
      setError(String(e));
      setSaving(false);
    }
  }

  return (
    <div className="min-h-dvh bg-muted/40">
      <header className="sticky top-0 z-30 border-b bg-background/85 pt-[env(safe-area-inset-top,0px)] backdrop-blur-md">
        <div className="flex h-14 items-center gap-1 px-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate(-1)}
            aria-label="Back"
          >
            <ChevronLeft className="size-5" />
          </Button>
          <div className="flex-1">
            <h1 className="text-[17px] leading-tight font-semibold tracking-tight">
              New item
            </h1>
            <p className="text-xs text-muted-foreground">
              {batchCount > 0
                ? `${batchCount} saved this session`
                : 'Draft · not saved'}
            </p>
          </div>
          <Button
            variant="ghost"
            className="text-muted-foreground"
            onClick={() => navigate('/inventory')}
          >
            {batchCount > 0 ? 'Done' : 'Cancel'}
          </Button>
        </div>
      </header>

      {photos.length > 0 && (
        <div className="border-b bg-background">
          <div className="flex gap-2 overflow-x-auto px-4 py-3">
            {previews.map((url, i) => (
              <div
                key={url}
                className="relative size-24 shrink-0 overflow-hidden rounded-md border"
              >
                <img src={url} alt="" className="size-full object-cover" />
                {i === 0 && (
                  <span className="absolute top-1 left-1 rounded bg-black/60 px-1.5 py-0.5 text-[10px] font-medium text-white">
                    Cover
                  </span>
                )}
                <button
                  type="button"
                  onClick={() => removePhoto(i)}
                  aria-label="Remove photo"
                  className="absolute top-1 right-1 rounded-full bg-black/60 p-0.5 text-white"
                >
                  <X className="size-3" />
                </button>
              </div>
            ))}
            <button
              type="button"
              onClick={() => libraryInput.current?.click()}
              aria-label="Add photos"
              className="flex size-24 shrink-0 items-center justify-center rounded-md border border-dashed bg-muted/50 text-muted-foreground"
            >
              <Plus className="size-5" />
            </button>
          </div>
        </div>
      )}

      <main className="space-y-4 px-4 pt-4 pb-32">
        {lastSaved && (
          <div className="flex items-center justify-between gap-3 rounded-md border bg-background px-3 py-2">
            <p className="min-w-0 truncate text-sm">
              Saved <span className="font-medium">{lastSaved.name}</span>
            </p>
            <button
              type="button"
              onClick={() => navigate(`/inventory/item/${lastSaved.id}`)}
              className="shrink-0 text-sm font-medium underline"
            >
              Open
            </button>
          </div>
        )}

        <Card>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold">Photos</h2>
              {photos.length > 0 && (
                <span className="rounded-full bg-secondary px-2 py-0.5 text-[11px] font-medium text-secondary-foreground">
                  {photos.length} staged
                </span>
              )}
            </div>
            <div className="flex gap-2">
              <Button
                className="h-11 flex-1"
                disabled={converting}
                onClick={() => cameraInput.current?.click()}
              >
                <Camera /> {converting ? 'Preparing…' : 'Take photo'}
              </Button>
              <Button
                variant="outline"
                className="h-11 flex-1"
                disabled={converting}
                onClick={() => libraryInput.current?.click()}
              >
                <Images /> Library
              </Button>
            </div>
            <input
              ref={cameraInput}
              type="file"
              // Not image/*: iOS hands over the original HEIC for that, which
              // the API rejects (422). Naming the types makes it convert to JPEG.
              accept={ACCEPTED_IMAGE_TYPES}
              capture="environment"
              hidden
              onChange={(e) => {
                void addPhotos(e.target.files);
                e.target.value = '';
              }}
            />
            <input
              ref={libraryInput}
              type="file"
              accept={ACCEPTED_IMAGE_TYPES}
              multiple
              hidden
              onChange={(e) => {
                void addPhotos(e.target.files);
                e.target.value = '';
              }}
            />
          </CardContent>
        </Card>

        <ItemFieldCards value={fields} onChange={setFields} idPrefix="new" />

        <ChipPicker
          title="Selling places"
          hint="Where this item will be listed"
          rows={places}
          checked={checkedPlaces}
          onToggle={(id) => toggle(checkedPlaces, id, setCheckedPlaces)}
          onAdd={addPlace}
          addPlaceholder="Add place…"
        />

        <ChipPicker
          title="Labels"
          hint="For searching later"
          rows={labels}
          checked={checkedLabels}
          onToggle={(id) => toggle(checkedLabels, id, setCheckedLabels)}
          onAdd={addLabel}
          addPlaceholder="Add label…"
        />

        <NotesCard value={fields} onChange={setFields} idPrefix="new" />

        {error && (
          <p className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {error}
          </p>
        )}
      </main>

      <div className="fixed inset-x-0 bottom-0 z-30 border-t bg-background/90 pb-[calc(1.5rem+env(safe-area-inset-bottom,0px))] backdrop-blur-md">
        <div className="px-4 pt-3">
          <Button
            className="h-12 w-full text-[15px]"
            onClick={save}
            disabled={saving}
          >
            {saving ? 'Saving…' : 'Save & add another'}
          </Button>
        </div>
      </div>
    </div>
  );
}
