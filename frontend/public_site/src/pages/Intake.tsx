import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import * as stylex from '@stylexjs/stylex'
import {
  createItem,
  createLabel,
  createSellingPlace,
  listLabels,
  listSellingPlaces,
  uploadImage,
  type ItemImage,
  type Label,
  type SellingPlace,
} from '../api/client'

const inputStyle = { width: '100%', padding: 8, boxSizing: 'border-box' } as const

export default function IntakePage() {
  const navigate = useNavigate()

  // fields
  const [name, setName] = useState('')
  const [cost, setCost] = useState('')
  const [purchasedAt, setPurchasedAt] = useState('')
  const [length_, setLength] = useState('')
  const [width, setWidth] = useState('')
  const [height, setHeight] = useState('')
  const [unit, setUnit] = useState<'inch' | 'cm'>('inch')
  const [extraMeasurements, setExtraMeasurements] = useState('')
  const [weightLbs, setWeightLbs] = useState('')
  const [weightOz, setWeightOz] = useState('')
  const [notes, setNotes] = useState('')
  const [whatnot, setWhatnot] = useState('')

  // lists
  const [places, setPlaces] = useState<SellingPlace[]>([])
  const [checkedPlaces, setCheckedPlaces] = useState<Set<string>>(new Set())
  const [newPlaceName, setNewPlaceName] = useState('')
  const [labels, setLabels] = useState<Label[]>([])
  const [checkedLabels, setCheckedLabels] = useState<Set<string>>(new Set())
  const [newLabelName, setNewLabelName] = useState('')

  // photos staged before save
  const [photos, setPhotos] = useState<File[]>([])
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listSellingPlaces().then(setPlaces).catch(() => {})
    listLabels().then(setLabels).catch(() => {})
  }, [])

  function toggle(set: Set<string>, id: string, apply: (s: Set<string>) => void) {
    const next = new Set(set)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    apply(next)
  }

  async function addPlace() {
    const name = newPlaceName.trim()
    if (!name) return
    try {
      const p = await createSellingPlace(name)
      setPlaces((prev) => [...prev, p])
      setCheckedPlaces((prev) => new Set(prev).add(p.id))
      setNewPlaceName('')
    } catch (e) {
      setError(String(e))
    }
  }

  async function addLabel() {
    const name = newLabelName.trim()
    if (!name) return
    try {
      const l = await createLabel(name)
      setLabels((prev) => [...prev, l])
      setCheckedLabels((prev) => new Set(prev).add(l.id))
      setNewLabelName('')
    } catch (e) {
      setError(String(e))
    }
  }

  async function save() {
    if (!name.trim()) {
      setError('Name is required')
      return
    }
    setSaving(true)
    setError(null)
    try {
      const item = await createItem({
        name: name.trim(),
        acquisition_cost_cents: cost ? Math.round(parseFloat(cost) * 100) : undefined,
        purchased_at: purchasedAt || undefined,
        length: length_ ? parseFloat(length_) : undefined,
        width: width ? parseFloat(width) : undefined,
        height: height ? parseFloat(height) : undefined,
        measurement_unit: unit,
        extra_measurements: extraMeasurements || undefined,
        weight_lbs: weightLbs ? parseInt(weightLbs, 10) : undefined,
        weight_oz: weightOz ? parseFloat(weightOz) : undefined,
        notes: notes || undefined,
        whatnot_number: whatnot || undefined,
        selling_place_ids: [...checkedPlaces],
        label_ids: [...checkedLabels],
      })
      // Upload staged photos sequentially; failures surface but keep saved item.
      const uploaded: ItemImage[] = []
      for (const file of photos) {
        try {
          uploaded.push(await uploadImage(item.id, file))
        } catch (e) {
          setError(`Item saved, but a photo failed to upload: ${String(e)}`)
        }
      }
      void uploaded
      navigate(`/item/${item.id}`)
    } catch (e) {
      setError(String(e))
      setSaving(false)
    }
  }

  return (
    <main {...stylex.props(styles.page)}>
      <h1>New item</h1>

      <label>Photos</label>
      <input
        type="file"
        accept="image/*"
        capture="environment"
        multiple
        onChange={(e) => {
          const files = Array.from(e.target.files ?? [])
          setPhotos((prev) => [...prev, ...files])
          e.target.value = ''
        }}
      />
      {photos.length > 0 && <p>{photos.length} photo(s) staged</p>}

      <label>Name *</label>
      <input value={name} onChange={(e) => setName(e.target.value)} style={inputStyle} />

      <label>Purchase price ($)</label>
      <input type="number" step="0.01" value={cost} onChange={(e) => setCost(e.target.value)} style={inputStyle} />

      <label>Purchase date</label>
      <input type="date" value={purchasedAt} onChange={(e) => setPurchasedAt(e.target.value)} style={inputStyle} />

      <label>Measurements (inches/cm)</label>
      <div {...stylex.props(styles.row3)}>
        <input type="number" placeholder="L" value={length_} onChange={(e) => setLength(e.target.value)} />
        <input type="number" placeholder="W" value={width} onChange={(e) => setWidth(e.target.value)} />
        <input type="number" placeholder="H" value={height} onChange={(e) => setHeight(e.target.value)} />
        <select value={unit} onChange={(e) => setUnit(e.target.value as 'inch' | 'cm')}>
          <option value="inch">in</option>
          <option value="cm">cm</option>
        </select>
      </div>
      <input
        placeholder="Extra measurements (waist, diameter…)"
        value={extraMeasurements}
        onChange={(e) => setExtraMeasurements(e.target.value)}
        style={inputStyle}
      />

      <label>Weight</label>
      <div {...stylex.props(styles.row2)}>
        <input type="number" placeholder="lbs" value={weightLbs} onChange={(e) => setWeightLbs(e.target.value)} />
        <input type="number" step="0.1" placeholder="oz" value={weightOz} onChange={(e) => setWeightOz(e.target.value)} />
      </div>

      <label>Notes</label>
      <textarea value={notes} onChange={(e) => setNotes(e.target.value)} style={inputStyle} />

      <label>Whatnot number</label>
      <input value={whatnot} onChange={(e) => setWhatnot(e.target.value)} style={inputStyle} />

      <label>Selling places</label>
      {places.map((p) => (
        <label key={p.id}>
          <input
            type="checkbox"
            checked={checkedPlaces.has(p.id)}
            onChange={() => toggle(checkedPlaces, p.id, setCheckedPlaces)}
          />{' '}
          {p.name}
        </label>
      ))}
      <div {...stylex.props(styles.addRow)}>
        <input
          placeholder="Add place…"
          value={newPlaceName}
          onChange={(e) => setNewPlaceName(e.target.value)}
          style={inputStyle}
        />
        <button type="button" onClick={addPlace}>Add</button>
      </div>

      <label>Labels</label>
      {labels.map((l) => (
        <label key={l.id}>
          <input
            type="checkbox"
            checked={checkedLabels.has(l.id)}
            onChange={() => toggle(checkedLabels, l.id, setCheckedLabels)}
          />{' '}
          {l.name}
        </label>
      ))}
      <div {...stylex.props(styles.addRow)}>
        <input
          placeholder="Add label…"
          value={newLabelName}
          onChange={(e) => setNewLabelName(e.target.value)}
          style={inputStyle}
        />
        <button type="button" onClick={addLabel}>Add</button>
      </div>

      {error && <p {...stylex.props(styles.error)}>{error}</p>}
      <button onClick={save} disabled={saving} {...stylex.props(styles.save)}>
        {saving ? 'Saving…' : 'Save item'}
      </button>
    </main>
  )
}

const styles = stylex.create({
  page: { padding: 16, maxWidth: 480, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 4 },
  row3: { display: 'flex', gap: 8 },
  row2: { display: 'flex', gap: 8 },
  addRow: { display: 'flex', gap: 8 },
  error: { color: '#b3261e' },
  save: { padding: 12, marginTop: 12 },
})