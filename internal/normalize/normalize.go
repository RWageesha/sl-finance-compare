// Package normalize maps each bank scraper's typed output
// (models.FixedDepositRate, SavingsRate, LoanRate) onto the shared
// products/product_rates schema — resolving or creating the right
// product_categories and products rows along the way. Scraper parsing
// logic stays untouched; this is purely the translation step between "raw
// scraped data" and "what goes in the database".
package normalize

import "strings"

// titleFromCode turns a category code like "SENIOR_CITIZEN_FD" into a
// readable fallback name ("Senior Citizen Fd"), used only when a category
// wasn't already seeded with a curated name by migration 004.
func titleFromCode(code string) string {
	words := strings.Split(strings.ToLower(code), "_")
	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}
