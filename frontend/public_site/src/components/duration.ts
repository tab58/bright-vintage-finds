// Compact elapsed-time labels for the inventory list: 134d, 8h, 42m.
// Days once past 24h, because that is the unit stock is judged in.
export function shortDuration(fromISO: string, toISO?: string): string {
  const from = new Date(fromISO).getTime()
  const to = toISO ? new Date(toISO).getTime() : Date.now()
  const ms = Math.max(0, to - from)

  const minutes = Math.floor(ms / 60_000)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h`
  return `${Math.floor(hours / 24)}d`
}
