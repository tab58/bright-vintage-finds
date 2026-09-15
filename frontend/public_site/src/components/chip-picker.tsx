import { useState } from 'react';
import { Check, Plus } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';

export type ChipRow = { id: string; name: string };

type Props = {
  title: string;
  hint: string;
  rows: ChipRow[];
  checked: Set<string>;
  onToggle: (id: string) => void;
  /** Omit to hide the add row (e.g. on a read-only item). */
  onAdd?: (name: string) => Promise<void>;
  addPlaceholder?: string;
  disabled?: boolean;
};

// Toggleable chips plus an inline add row — used for selling places and labels
// on both the intake and item pages.
export function ChipPicker({
  title,
  hint,
  rows,
  checked,
  onToggle,
  onAdd,
  addPlaceholder = 'Add…',
  disabled = false,
}: Props) {
  const [draft, setDraft] = useState('');

  async function add() {
    const name = draft.trim();
    if (!name || !onAdd) return;
    await onAdd(name);
    setDraft('');
  }

  return (
    <Card>
      <CardContent className="space-y-3">
        <div>
          <h2 className="text-sm font-semibold">{title}</h2>
          <p className="mt-0.5 text-xs text-muted-foreground">{hint}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {/* A locked item is a record: show what it has, not what it could have. */}
          {disabled && checked.size === 0 && (
            <span className="text-[15px] text-muted-foreground">—</span>
          )}
          {rows
            .filter((row) => !disabled || checked.has(row.id))
            .map((row) => {
              const on = checked.has(row.id);
              return (
                <button
                  key={row.id}
                  type="button"
                  aria-pressed={on}
                  disabled={disabled}
                  onClick={() => onToggle(row.id)}
                  className={
                    'inline-flex items-center gap-1.5 rounded-full border px-3 py-1.5 text-[13px] font-medium transition-colors ' +
                    (disabled
                      ? 'bg-muted text-foreground'
                      : on
                        ? 'border-primary bg-primary text-primary-foreground'
                        : 'bg-background')
                  }
                >
                  {on && !disabled && (
                    <Check className="size-3.5" strokeWidth={3} />
                  )}
                  {row.name}
                </button>
              );
            })}
        </div>
        {onAdd && !disabled && (
          <div className="flex gap-2">
            <Input
              className="h-10"
              placeholder={addPlaceholder}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault();
                  void add();
                }
              }}
            />
            <Button
              type="button"
              variant="outline"
              size="icon"
              className="size-10"
              onClick={() => void add()}
              aria-label={addPlaceholder}
            >
              <Plus />
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
