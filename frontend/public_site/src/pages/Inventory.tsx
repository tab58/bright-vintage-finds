import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import * as stylex from '@stylexjs/stylex'
import { listItems, listLabels, listSellingPlaces, type Item, type Label, type SellingPlace } from '../api/client'

export default function InventoryPage() {
  const [items, setItems] = useState<Item[]>([])
  const [places, setPlaces] = useState<SellingPlace[]>([])
  const [labels, setLabels] = useState<Label[]>([])
  const [query, setQuery] = useState('')
  const [placeId, setPlaceId] = useState('')
  const [labelId, setLabelId] = useState('')
  const [whatnot, setWhatnot] = useState('')
  const [status, setStatus] = useState('')
  const [error, setError] = useState<string | null>(null)

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
  }, [query, placeId, labelId, whatnot, status])

  return (
    <main {...stylex.props(styles.page)}>
      <h1>Inventory</h1>
      <div {...stylex.props(styles.filters)}>
        <input
          placeholder="Search name…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <select value={placeId} onChange={(e) => setPlaceId(e.target.value)}>
          <option value="">Any place</option>
          {places.map((p) => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
        <select value={labelId} onChange={(e) => setLabelId(e.target.value)}>
          <option value="">Any label</option>
          {labels.map((l) => (
            <option key={l.id} value={l.id}>{l.name}</option>
          ))}
        </select>
        <input
          placeholder="Whatnot #"
          value={whatnot}
          onChange={(e) => setWhatnot(e.target.value)}
        />
        <select value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="">Any status</option>
          <option value="draft">Draft</option>
          <option value="listed">Listed</option>
          <option value="sold">Sold</option>
          <option value="archived">Archived</option>
        </select>
      </div>
      {error && <p {...stylex.props(styles.error)}>{error}</p>}
      <ul {...stylex.props(styles.list)}>
        {items.map((it) => (
          <li key={it.id} {...stylex.props(styles.row)}>
            <Link to={`/item/${it.id}`} {...stylex.props(styles.link)}>
              {it.name}
            </Link>
            <span {...stylex.props(styles.meta)}>
              {it.status}
              {it.whatnot_number ? ` · WN ${it.whatnot_number}` : ''}
              {it.sold_price_cents != null ? ` · sold $${(it.sold_price_cents / 100).toFixed(2)}` : ''}
            </span>
          </li>
        ))}
      </ul>
      {items.length === 0 && !error && <p>No items match.</p>}
    </main>
  )
}

const styles = stylex.create({
  page: { padding: 16, maxWidth: 640, margin: '0 auto' },
  filters: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: 8,
    marginBottom: 12,
  },
  list: { listStyle: 'none', padding: 0 },
  row: {
    display: 'flex',
    justifyContent: 'space-between',
    padding: '10px 4px',
    borderBottom: '1px solid #e5ddd3',
  },
  link: { color: '#3b2f2a', fontWeight: 600, textDecoration: 'none' },
  meta: { color: '#7a6f63', fontSize: 13 },
  error: { color: '#b3261e' },
})