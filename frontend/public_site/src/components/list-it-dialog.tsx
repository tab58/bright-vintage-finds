import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';

// Asking for the price is the point of this dialog: an item goes on the shop
// front the moment it is listed, so it cannot go out without one. Shared by the
// item detail page and the inventory list's long-press move.
export function ListItDialog({
  open,
  onOpenChange,
  itemName,
  defaultPriceCents,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  itemName: string;
  /** Prefill, for a relist or a price already typed at intake. */
  defaultPriceCents?: number;
  onConfirm: (priceCents: number) => Promise<void>;
}) {
  const [price, setPrice] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setPrice(
        defaultPriceCents != null ? (defaultPriceCents / 100).toFixed(2) : '',
      );
      setError(null);
    }
  }, [open, defaultPriceCents]);

  const cents = Math.round(parseFloat(price) * 100);
  const ready = Number.isFinite(cents) && cents > 0;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[420px]">
        <DialogHeader>
          <DialogTitle>List it</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <p className="text-sm text-muted-foreground">
            “{itemName}” goes on the shop front at this price.
          </p>
          <div className="space-y-2">
            <Label htmlFor="list-price">List price</Label>
            <div className="relative">
              <span className="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground">
                $
              </span>
              <Input
                id="list-price"
                className="h-11 pl-7"
                inputMode="decimal"
                placeholder="0.00"
                value={price}
                onChange={(e) => setPrice(e.target.value)}
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
              setSaving(true);
              try {
                await onConfirm(cents);
              } catch (e) {
                setError(String(e));
              } finally {
                setSaving(false);
              }
            }}
          >
            {saving ? 'Listing…' : 'List it'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
