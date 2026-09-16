import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import type { Item, ItemBody } from '../api/client';

// The item form, shared by intake and the item detail page so both show the
// same fields in the same order. Values are kept as strings, the way the
// inputs hand them over; conversion happens in fieldsToBody.
export type ItemFields = {
  name: string;
  cost: string;
  listPrice: string;
  purchasedAt: string;
  length: string;
  width: string;
  height: string;
  unit: 'inch' | 'cm';
  extraMeasurements: string;
  weightLbs: string;
  weightOz: string;
  notes: string;
  whatnot: string;
};

export const emptyFields: ItemFields = {
  name: '',
  cost: '',
  listPrice: '',
  purchasedAt: '',
  length: '',
  width: '',
  height: '',
  unit: 'inch',
  extraMeasurements: '',
  weightLbs: '',
  weightOz: '',
  notes: '',
  whatnot: '',
};

const num = (v?: number) => (v != null ? String(v) : '');

export function fieldsFromItem(it: Item): ItemFields {
  return {
    name: it.name,
    cost:
      it.acquisition_cost_cents != null
        ? (it.acquisition_cost_cents / 100).toFixed(2)
        : '',
    listPrice:
      it.listing_price_cents != null
        ? (it.listing_price_cents / 100).toFixed(2)
        : '',
    purchasedAt: it.purchased_at?.slice(0, 10) ?? '',
    length: num(it.length),
    width: num(it.width),
    height: num(it.height),
    unit: it.measurement_unit,
    extraMeasurements: it.extra_measurements ?? '',
    weightLbs: num(it.weight_lbs),
    weightOz: num(it.weight_oz),
    notes: it.notes ?? '',
    whatnot: it.whatnot_number ?? '',
  };
}

// Text fields are sent even when empty: the API reads "" as "clear it".
// Numbers and dates drop out instead, since there is no cleared number.
export function fieldsToBody(f: ItemFields): ItemBody {
  return {
    name: f.name.trim(),
    acquisition_cost_cents: f.cost
      ? Math.round(parseFloat(f.cost) * 100)
      : undefined,
    // ponytail: emptying the field leaves the stored price alone rather than
    // clearing it, same as `Paid` — the API reads an absent number as "don't
    // touch". Clearing needs an empty-string convention like applyItemClears.
    listing_price_cents: f.listPrice
      ? Math.round(parseFloat(f.listPrice) * 100)
      : undefined,
    purchased_at: f.purchasedAt
      ? new Date(f.purchasedAt).toISOString()
      : undefined,
    length: f.length ? parseFloat(f.length) : undefined,
    width: f.width ? parseFloat(f.width) : undefined,
    height: f.height ? parseFloat(f.height) : undefined,
    measurement_unit: f.unit,
    extra_measurements: f.extraMeasurements,
    weight_lbs: f.weightLbs ? parseInt(f.weightLbs, 10) : undefined,
    weight_oz: f.weightOz ? parseFloat(f.weightOz) : undefined,
    notes: f.notes,
    whatnot_number: f.whatnot,
  };
}

type Props = {
  value: ItemFields;
  onChange: (next: ItemFields) => void;
  disabled?: boolean;
  /** Prefix for input ids, so two forms can coexist on one page. */
  idPrefix?: string;
};

export function ItemFieldCards({
  value,
  onChange,
  disabled = false,
  idPrefix = 'f',
}: Props) {
  const set = <K extends keyof ItemFields>(key: K, v: ItemFields[K]) =>
    onChange({ ...value, [key]: v });
  const id = (name: string) => `${idPrefix}-${name}`;

  return (
    <>
      <Card>
        <CardContent className="space-y-4">
          <h2 className="text-sm font-semibold">Details</h2>

          <div className="space-y-2">
            <Label htmlFor={id('name')}>
              Name <span className="text-destructive">*</span>
            </Label>
            <Input
              id={id('name')}
              className="h-11"
              placeholder="1970s Pyrex mixing bowl"
              value={value.name}
              disabled={disabled}
              onChange={(e) => set('name', e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-2">
              <Label htmlFor={id('cost')}>Paid</Label>
              <div className="relative">
                <span className="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground">
                  $
                </span>
                <Input
                  id={id('cost')}
                  className="h-11 pl-7"
                  inputMode="decimal"
                  placeholder="0.00"
                  value={value.cost}
                  disabled={disabled}
                  onChange={(e) => set('cost', e.target.value)}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor={id('list-price')}>List price</Label>
              <div className="relative">
                <span className="pointer-events-none absolute top-1/2 left-3 -translate-y-1/2 text-muted-foreground">
                  $
                </span>
                <Input
                  id={id('list-price')}
                  className="h-11 pl-7"
                  inputMode="decimal"
                  placeholder="0.00"
                  value={value.listPrice}
                  disabled={disabled}
                  onChange={(e) => set('listPrice', e.target.value)}
                />
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor={id('purchased')}>Purchased</Label>
            <Input
              id={id('purchased')}
              type="date"
              className="h-11"
              value={value.purchasedAt}
              disabled={disabled}
              onChange={(e) => set('purchasedAt', e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor={id('whatnot')}>Whatnot number</Label>
            <Input
              id={id('whatnot')}
              className="h-11"
              inputMode="numeric"
              placeholder="e.g. 214"
              value={value.whatnot}
              disabled={disabled}
              onChange={(e) => set('whatnot', e.target.value)}
            />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold">Measurements</h2>
            <ToggleGroup
              type="single"
              value={value.unit}
              onValueChange={(v) => v && set('unit', v as 'inch' | 'cm')}
              variant="outline"
              size="sm"
              disabled={disabled}
            >
              <ToggleGroupItem value="inch" aria-label="Inches">
                in
              </ToggleGroupItem>
              <ToggleGroupItem value="cm" aria-label="Centimeters">
                cm
              </ToggleGroupItem>
            </ToggleGroup>
          </div>

          <div className="grid grid-cols-3 gap-3">
            {(
              [
                ['length', 'Length'],
                ['width', 'Width'],
                ['height', 'Height'],
              ] as const
            ).map(([key, label]) => (
              <div key={key} className="space-y-1.5">
                <Label
                  htmlFor={id(key)}
                  className="text-xs text-muted-foreground"
                >
                  {label}
                </Label>
                <Input
                  id={id(key)}
                  className="h-11 text-center"
                  inputMode="decimal"
                  placeholder="—"
                  value={value[key]}
                  disabled={disabled}
                  onChange={(e) => set(key, e.target.value)}
                />
              </div>
            ))}
          </div>

          <div className="space-y-2">
            <Label htmlFor={id('extra')}>Extra measurements</Label>
            <Input
              id={id('extra')}
              className="h-11"
              placeholder="Waist, diameter, rim…"
              value={value.extraMeasurements}
              disabled={disabled}
              onChange={(e) => set('extraMeasurements', e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label
                htmlFor={id('lbs')}
                className="text-xs text-muted-foreground"
              >
                Weight (lbs)
              </Label>
              <Input
                id={id('lbs')}
                className="h-11 text-center"
                inputMode="numeric"
                placeholder="0"
                value={value.weightLbs}
                disabled={disabled}
                onChange={(e) => set('weightLbs', e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label
                htmlFor={id('oz')}
                className="text-xs text-muted-foreground"
              >
                Weight (oz)
              </Label>
              <Input
                id={id('oz')}
                className="h-11 text-center"
                inputMode="decimal"
                placeholder="0.0"
                value={value.weightOz}
                disabled={disabled}
                onChange={(e) => set('weightOz', e.target.value)}
              />
            </div>
          </div>
        </CardContent>
      </Card>
    </>
  );
}

export function NotesCard({
  value,
  onChange,
  disabled = false,
  idPrefix = 'f',
}: Props) {
  return (
    <Card>
      <CardContent className="space-y-2">
        <Label htmlFor={`${idPrefix}-notes`}>Notes</Label>
        <Textarea
          id={`${idPrefix}-notes`}
          rows={3}
          placeholder="Condition, flaws, provenance…"
          value={value.notes}
          disabled={disabled}
          onChange={(e) => onChange({ ...value, notes: e.target.value })}
        />
      </CardContent>
    </Card>
  );
}

// Read-only rendering of the same fields, for the item detail page: intake
// captures them, the detail page only shows them until Edit details is on.
function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-4 border-b py-2 last:border-b-0">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-right text-[15px]">{value || '—'}</span>
    </div>
  );
}

export function ItemSummaryCards({ value }: { value: ItemFields }) {
  const dims = [value.length, value.width, value.height].filter(Boolean);
  const size =
    dims.length > 0
      ? `${dims.join(' × ')} ${value.unit === 'inch' ? 'in' : 'cm'}`
      : '';
  const weight = [
    value.weightLbs && `${value.weightLbs} lb`,
    value.weightOz && `${value.weightOz} oz`,
  ]
    .filter(Boolean)
    .join(' ');

  return (
    <Card>
      <CardContent className="space-y-0">
        <Row label="Paid" value={value.cost ? `$${value.cost}` : ''} />
        <Row
          label="List price"
          value={value.listPrice ? `$${value.listPrice}` : ''}
        />
        <Row label="Purchased" value={value.purchasedAt} />
        <Row label="Size" value={size} />
        <Row label="Extra measurements" value={value.extraMeasurements} />
        <Row label="Weight" value={weight} />
      </CardContent>
    </Card>
  );
}
