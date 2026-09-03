// Small formatting helpers shared by pages/components. Nuxt auto-imports
// everything exported from app/utils/, so these are used with no import
// statement needed. Ported from the fmtDate/fmtTenure/formatCategoryLabel
// functions in the old web/rates.html and web/index.html.

export function fmtDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// A relative "N days/hours ago" label, used by the homepage's "Recently
// Updated" table.
export function fmtRelativeDate(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const hours = Math.round((Date.now() - d.getTime()) / 3_600_000)
  if (hours < 1) return 'just now'
  if (hours < 24) return `${hours} hour${hours === 1 ? '' : 's'} ago`
  const days = Math.round(hours / 24)
  return `${days} day${days === 1 ? '' : 's'} ago`
}

export function fmtTenure(months: number): string {
  if (months % 12 === 0 && months >= 12) {
    const years = months / 12
    return `${years} ${years === 1 ? 'year' : 'years'}`
  }
  return `${months} ${months === 1 ? 'month' : 'months'}`
}

// Category codes are stored as e.g. "SENIOR_CITIZEN_FD" — display them as
// "Senior Citizen Fd" rather than the raw underscored code.
export function formatCategoryLabel(code: string): string {
  return String(code)
    .toLowerCase()
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase())
}
