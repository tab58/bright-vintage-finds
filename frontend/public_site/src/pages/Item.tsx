import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import * as stylex from '@stylexjs/stylex'
import {
  getItem,
  listSellingPlaces,
  markItemSold,
  updateItem,
  type Item,
  type SellingPlace,
} from '../api/client'

export default function ItemPage() {
  const { id } = useParams()
  const [item, setItem] = useState<Item | null>(null)
  const [places, setPlaces] = useState<SellingPlace[]>([])
  const [error, setError] = useState<string | null>(null)

  // mark-sold form
  const [soldAt, setSoldAt] = useState(new Date().toISOString().slice(0, 10))
  const [soldPrice, setSoldPrice] = useState('')
  const [soldPlaceId, setSoldPlaceId] = useState('')

  // edit
  const [name, setName] = useState('')
  const [cost, setCost] = useState('')
  const [notes, setNotes] = useState('')
  const [whatnot, setWhatnot] = useState('')

  useEffect(() => {
    if (!id) return
    getItem(id).then(setItem).catch((e) => setError(String(e)))
    listSellingPlaces().then(setPlaces).catch(() => {})
  }, [id])

  useEffect(() => {
    // Default the sold-platform picker to the item's first listed place.
    if (item && soldPlaceId === '' && item.selling_places.length > 0) {
      const match = places.find((p) => p.name === item.selling_places[0])
      if (match) setSoldPlaceId(match.id)
    }
  }, [item, places, soldPlaceId])

  async function saveEdits() {
    if (!item) return
    setError(null)
    try {
      const updated = await updateItem(item.id, {
        name,
        acquisition_cost_cents: cost ? Math.round(parseFloat(cost) * 100) : undefined,
        notes: notes || undefined,
        whatnot_number: whatnot || undefined,
      })
      setItem(updated)
    } catch (e) {
      setError(String(e))
    }
  }

  async function markSold() {
    if (!item || !soldPlaceId) {
      setError('Pick the platform where it sold')
      return
    }
    setError(null)
    try {
      const updated = await markItemSold(item.id, {
        sold_at: new Date(soldAt).toISOString(),
        sold_price_cents: Math.round(parseFloat(soldPrice) * 100),
        sold_place_id: soldPlaceId,
      })
      setItem(updated)
    } catch (e) {
      setError(String(e))
    }
  }

  if (error && !item) {
    return (
      <main {...stylex.props(styles.page)}>
        <p {...stylex.props(styles.error)}>{error}</p>
        <Link to="/">← Back</Link>
      </main>
    )
  }
  if (!item) {
    return <main {...stylex.props(styles.page)}><p>Loading…</p></main>
  }

  return (
    <main {...stylex.props(styles.page)}>
      <Link to="/">← Inventory</Link>
      <h1>{item.name}</h1>

      {item.images.length > 0 && (
        <div {...stylex.props(styles.photos)}>
          {item.images.map((img) => (
            <img key={img.id} src={img.url} alt={item.name} {...stylex.props(styles.photo)} />
          ))}
        </div>
      )}

      <p>
        Status: <strong>{item.status}</strong>
        {item.sold_price_cents != null && (
          <>
            {' '}· sold for ${(item.sold_price_cents / 100).toFixed(2)} on{' '}
            {item.sold_at?.slice(0, 10)} via {item.sold_place}
          </>
        )}
      </p>

      <label>Edit name</label>
      <input value={name} onChange={(e) => setName(e.target.value)} style={inputStyle} />
      <label>Purchase price ($)</label>
      <input type="number" step="0.01" value={cost} onChange={(e) => setCost(e.target.value)} style={inputStyle} />
      <label>Whatnot number</label>
      <input value={whatnot} onChange={(e) => setWhatnot(e.target.value)} style={inputStyle} />
      <label>Notes</label>
      <textarea value={notes} onChange={(e) => setNotes(e.target.value)} style={inputStyle} />
      <button onClick={saveEdits} {...stylex.props(styles.button)}>Save edits</button>

      {item.status !== 'sold' && (
        <section {...stylex.props(styles.soldBox)}>
          <h2>Mark sold</h2>
          <label>Date sold</label>
          <input type="date" value={soldAt} onChange={(e) => setSoldAt(e.target.value)} style={inputStyle} />
          <label>Sale price ($)</label>
          <input type="number" step="0.01" value={soldPrice} onChange={(e) => setSoldPrice(e.target.value)} style={inputStyle} />
          <label>Platform</label>
          <select value={soldPlaceId} onChange={(e) => setSoldPlaceId(e.target.value)} style={inputStyle}>
            <option value="">Choose…</option>
            {places.map((p) => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>
          <button onClick={markSold} {...stylex.props(styles.button)}>Mark as sold</button>
        </section>
      )}

      {error && <p {...stylex.props(styles.error)}>{error}</p>}
    </main>
  )
}

const inputStyle = { width: '100%', padding: 8, boxSizing: 'border-box' } as const

const styles = stylex.create({
  page: { padding: 16, maxWidth: 480, margin: '0 auto', display: 'flex', flexDirection: 'column', gap: 8 },
  photos: { display: 'flex', gap: 8, overflowX: 'auto' },
  photo: { width: 120, height: 120, objectFit: 'cover', borderRadius: 8 },
  soldBox: { borderTop: '1px solid #e5ddd3', paddingTop: 12, display: 'flex', flexDirection: 'column', gap: 8 },
  button: { padding: 10 },
  error: { color: '#b3261e' },
})