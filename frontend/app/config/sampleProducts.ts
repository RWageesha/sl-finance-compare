// Resolves a "View Details" click from any Product Directory or Compare
// row back to one product, by id, for the /products/sample/[id] page.
// Directory and Compare rows share ids for the same logical product, so
// this looks in both places and merges what it finds — this is the one
// canonical detail record per id, not a third hand-written copy.
import { PRODUCT_TYPES } from './productTypes'
import type { SampleRow } from './productTypes'

export interface ResolvedSampleProduct extends SampleRow {
  type: string
  typeLabel: string
  typeHref: string
  /** Only set when the directory and compare rows disagree on `rate`
   * (housing loans: a disclosed AWPLR-linked rate vs. a flat rate used
   * only to compute an EMI) — the flat one, kept separate instead of
   * silently overwriting the bank's real disclosed rate. */
  illustrativeRate?: string
}

export function findSampleProduct(id: string): ResolvedSampleProduct | null {
  for (const type of Object.values(PRODUCT_TYPES)) {
    const directoryRow = type.rows.find((r) => r.id === id)
    const compareRow = type.compare?.rows.find((r) => r.id === id)
    if (!directoryRow && !compareRow) continue
    const merged = {
      ...(compareRow ?? {}),
      ...(directoryRow ?? {}),
      id,
      type: type.slug,
      typeLabel: type.switcherLabel,
      typeHref: `/products/${type.slug}`
    } as ResolvedSampleProduct
    if (directoryRow && compareRow && directoryRow.rate !== compareRow.rate) {
      merged.rate = directoryRow.rate
      merged.illustrativeRate = compareRow.rate
      merged.rateValue = compareRow.rateValue
      merged.rateType = compareRow.rateType
    }
    return merged
  }
  return null
}
