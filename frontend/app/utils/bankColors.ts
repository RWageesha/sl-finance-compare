// Shared bank -> avatar color mapping, used everywhere a bank shows as a
// colored circular initial (product directory tables, recent-rates rows,
// search results, the Data page's chart, etc.) so the same bank always
// reads as the same color across the site — regardless of whether the
// caller has the real API's full legal name ("Commercial Bank of
// Ceylon") or the short form used in the Compare pages' placeholder rows
// ("Commercial Bank"). Both resolve through bankKey() to one canonical
// key before either lookup below, so a real bank's color/initial never
// silently falls back to the generic gray just because the caller had a
// different (but equally valid) spelling of its name.
const BANK_KEYS: Record<string, string> = {
  hnb: 'hnb',
  'hatton national bank': 'hnb',
  'commercial bank': 'combank',
  'commercial bank of ceylon': 'combank',
  'sampath bank': 'sampath',
  'bank of ceylon': 'boc',
  boc: 'boc',
  'seylan bank': 'seylan',
  "people's bank": 'peoples',
  'national savings bank': 'nsb',
  nsb: 'nsb',
  ndb: 'ndb',
  'ndb bank': 'ndb',
  'national development bank': 'ndb'
}

// Canonical key for a bank name in ANY form this app uses for it — lets
// a query-string prefill (or a color lookup) from one page match a row
// written in another page's naming convention.
export function bankKey(bankName: string): string {
  return BANK_KEYS[bankName.trim().toLowerCase()] ?? bankName.trim().toLowerCase()
}

const BANK_COLORS: Record<string, string> = {
  hnb: '#991B1B',
  combank: '#2563EB',
  sampath: '#EA580C',
  boc: '#A16207',
  seylan: '#16A34A',
  peoples: '#4F46E5',
  nsb: '#7C3AED',
  ndb: '#0EA5E9'
}

const FALLBACK_COLOR = '#64748B'

export function bankColor(bankName: string): string {
  return BANK_COLORS[bankKey(bankName)] ?? FALLBACK_COLOR
}

export function bankInitial(bankName: string): string {
  return bankName.trim().charAt(0).toUpperCase() || '?'
}
