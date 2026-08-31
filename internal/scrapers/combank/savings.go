package combank

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/wageesha/sl-finance-compare/internal/models"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/ratetext"
)

// interestRatesTabSelector is the tab-pane on the rates & tariff page that
// bundles Savings Account, Fixed Deposit, and foreign-currency products —
// only the savings ones are wanted here (FD is scraped from its own,
// cleaner page; FX is out of scope, LKR-only like every other scraper).
const interestRatesTabSelector = "#interest-rates"

// savingsSkipKeywords identifies expand-blocks under the interest-rates
// tab that are not savings accounts (Fixed Deposit / foreign-currency
// products living in the same tab).
var savingsSkipKeywords = []string{"fixed deposit", "foreign currency", "fc plus", "pfc", "bfc", "forex"}

// ParseSavings extracts savings account rates from ComBank's rates &
// tariff page. Every product under the "Interest Rates" tab is an
// "expand-block": a product name (.expand-link) followed by a single rate
// table (td[0] = row label, td[1] = rate, td[2] = optional AER, ignored —
// same shape the Fixed Deposit table on ComBank's own page uses). One
// product, "Udara Senior Citizens Account", bundles a savings row and two
// Fixed Deposit rows under the same block; rows labelled "fixed deposit"
// are skipped so they don't leak into savings data.
func ParseSavings(html []byte) ([]models.SavingsRate, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("combank: parse html: %w", err)
	}

	var (
		rates    []models.SavingsRate
		rowErrs  []string
		scrapeAt = time.Now().UTC()
	)

	doc.Find(interestRatesTabSelector + " .expand-block").Each(func(i int, block *goquery.Selection) {
		accountName := strings.TrimSpace(block.Find(".expand-link").First().Text())
		if accountName == "" {
			return
		}
		lowerName := strings.ToLower(accountName)
		for _, kw := range savingsSkipKeywords {
			if strings.Contains(lowerName, kw) {
				return
			}
		}

		block.Find("table tbody tr").Each(func(j int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() < 2 {
				return
			}

			label := strings.TrimSpace(cells.Eq(0).Text())
			if label == "" {
				return
			}
			if strings.Contains(strings.ToLower(label), "fixed deposit") {
				return // e.g. Udara Senior Citizens Account's bundled FD rows
			}

			balanceTier := label
			lowerLabel := strings.ToLower(label)
			if strings.Contains(lowerLabel, "interest paid") || strings.Contains(lowerLabel, "interest at") ||
				strings.Contains(lowerLabel, "int paid") || strings.Contains(lowerLabel, "int at") {
				balanceTier = "" // payout-frequency boilerplate (incl. ComBank's "Int paid..." abbreviation), not a real tier
			}

			rate, err := ratetext.ParseFlatRate(cells.Eq(1).Text())
			if err != nil {
				rowErrs = append(rowErrs, fmt.Sprintf("%s row %d: rate %q: %v", accountName, j, cells.Eq(1).Text(), err))
				return
			}

			rates = append(rates, models.SavingsRate{
				AccountName:  accountName,
				BalanceTier:  balanceTier,
				InterestRate: rate,
				SourceURL:    ratesTariffURL,
				ScrapedAt:    scrapeAt,
			})
		})
	})

	if len(rates) == 0 {
		detail := "no savings rows found"
		if len(rowErrs) > 0 {
			detail = strings.Join(rowErrs, "; ")
		}
		return nil, fmt.Errorf("combank: no savings rates parsed from rates-tariff page (selectors likely stale): %s", detail)
	}

	if len(rowErrs) > 0 {
		return rates, fmt.Errorf("combank: parsed %d savings rate(s) but skipped %d malformed row(s): %s",
			len(rates), len(rowErrs), strings.Join(rowErrs, "; "))
	}

	return rates, nil
}
