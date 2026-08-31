package hnb

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wageesha/sl-finance-compare/internal/models"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/ratetext"
)

const (
	savingsAccountsSubCategory    = "Savings Accounts Interest Rates"
	moneyMarketSavingsSubCategory = "Money Market Savings"
)

// savingsColumnMap describes, per division title, which column holds the
// account name (when it's not the division title itself), which holds the
// balance tier (empty string means "no tier column"), and which holds the
// comparable headline rate. Every division also publishes an "Effective
// Annual Rate" column that is intentionally not captured, same as ComBank's
// AER column — one comparable headline rate per row is enough.
type savingsColumnSpec struct {
	accountNameCol int // -1 means "use the division title instead"
	balanceTierCol int // -1 means "no balance tier for this division"
	rateCol        int
}

var savingsColumnSpecs = map[string]savingsColumnSpec{
	"HNB Savings +":                   {accountNameCol: -1, balanceTierCol: 0, rateCol: 1},
	"HNB FIT Account":                 {accountNameCol: -1, balanceTierCol: 1, rateCol: 2},
	"Savings Accounts Interest Rates": {accountNameCol: 0, balanceTierCol: -1, rateCol: 1},
	"Money Market Savings Rates":      {accountNameCol: 0, balanceTierCol: -1, rateCol: 1},
}

// ParseSavingsAccounts extracts savings account rate tiers from the raw
// JSON body returned by HNB's interest rates API. It looks at the "Savings
// Accounts Interest Rates" and "Money Market Savings" sub-categories.
func ParseSavingsAccounts(jsonBody []byte) ([]models.SavingsRate, error) {
	var resp apiResponse
	if err := json.Unmarshal(jsonBody, &resp); err != nil {
		return nil, fmt.Errorf("hnb: decode api response: %w", err)
	}

	var (
		rates    []models.SavingsRate
		rowErrs  []string
		scrapeAt = time.Now().UTC()
	)

	for _, subName := range []string{savingsAccountsSubCategory, moneyMarketSavingsSubCategory} {
		sub := findSubCategory(resp, subName)
		if sub == nil {
			rowErrs = append(rowErrs, fmt.Sprintf("sub-category %q not found", subName))
			continue
		}

		for _, div := range sub.SubCategoryDivisionApproved {
			title := strings.TrimSpace(div.Title)
			spec, ok := savingsColumnSpecs[title]
			if !ok {
				rowErrs = append(rowErrs, fmt.Sprintf("division %q: no column mapping known", title))
				continue
			}

			for _, td := range div.TableDataApproved {
				var payload apiTablePayload
				if err := json.Unmarshal([]byte(td.Data), &payload); err != nil {
					rowErrs = append(rowErrs, fmt.Sprintf("division %q: decode table data: %v", title, err))
					continue
				}

				for i, row := range payload.Rows {
					maxCol := spec.rateCol
					if spec.accountNameCol > maxCol {
						maxCol = spec.accountNameCol
					}
					if spec.balanceTierCol > maxCol {
						maxCol = spec.balanceTierCol
					}
					if len(row) <= maxCol {
						rowErrs = append(rowErrs, fmt.Sprintf("division %q row %d: too few columns", title, i))
						continue
					}

					accountName := title
					if spec.accountNameCol >= 0 {
						accountName = strings.TrimSpace(row[spec.accountNameCol])
					}
					if accountName == "" {
						continue // stray/empty row
					}

					balanceTier := ""
					if spec.balanceTierCol >= 0 {
						balanceTier = strings.TrimSpace(row[spec.balanceTierCol])
					}

					rate, err := ratetext.ParseFlatRate(row[spec.rateCol])
					if err != nil {
						continue // e.g. the 0.0% / "-" zero-balance tier
					}

					rates = append(rates, models.SavingsRate{
						AccountName:  accountName,
						BalanceTier:  balanceTier,
						InterestRate: rate,
						SourceURL:    ratesAPIURL,
						ScrapedAt:    scrapeAt,
					})
				}
			}
		}
	}

	if len(rates) == 0 {
		detail := "no savings account rows found"
		if len(rowErrs) > 0 {
			detail = strings.Join(rowErrs, "; ")
		}
		return nil, fmt.Errorf("hnb: no savings rates parsed from api response (schema likely changed): %s", detail)
	}

	if len(rowErrs) > 0 {
		return rates, fmt.Errorf("hnb: parsed %d savings rate(s) but skipped %d malformed row(s)/division(s): %s",
			len(rates), len(rowErrs), strings.Join(rowErrs, "; "))
	}

	return rates, nil
}
