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

// lendingRatesTabSelector is the tab-pane on the rates & tariff page
// covering loan products (Lease Facilities, Personal Loans, Home Loans,
// Gold Loans Pawning, Diribala Loans, Agriculture Sector, All Other
// Advances, Penal Interest on Overdue Credit Facilities).
const lendingRatesTabSelector = "#lending-rates"

// ParseLoans extracts loan rates from ComBank's rates & tariff page.
// Each product is an "expand-block" (.expand-link name + a table) in one
// of two shapes:
//   - 2-column: Description -> Interest Rate (Per Annum). td[0] is the
//     row's own label (e.g. "Short Term Gold Loans -03 Months").
//   - matrix: a tier/type column (td[0], e.g. "Standard"/"Premium") plus
//     one rate column per tenure ("1 Years" ... "6-7 Years").
//
// td[0] is always treated as the row label (-> RateLabel); every other
// column is a rate candidate, paired with the table's header text at the
// same offset counting from the right (handles the matrix header row
// being one cell short of the data row, since its leading rowspan cell
// isn't repeated). Tenure is left blank for 2-column tables, since there
// the "header" is just "Interest Rate (Per Annum)", not a real tenure.
// Cells that aren't a single flat percentage — floating formulas
// ("AWPLR + 2.50%"), "-", "N/A", commission add-ons — fail
// ratetext.ParseFlatRate and are skipped without special-casing.
func ParseLoans(html []byte) ([]models.LoanRate, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("combank: parse html: %w", err)
	}

	var (
		rates    []models.LoanRate
		rowErrs  []string
		scrapeAt = time.Now().UTC()
	)

	doc.Find(lendingRatesTabSelector + " .expand-block").Each(func(i int, block *goquery.Selection) {
		productName := strings.TrimSpace(block.Find(".expand-link").First().Text())
		if productName == "" {
			return
		}

		table := block.Find("table").First()
		if table.Length() == 0 {
			return
		}

		var headers []string
		table.Find("thead tr").Last().Find("th").Each(func(_ int, th *goquery.Selection) {
			headers = append(headers, strings.TrimSpace(th.Text()))
		})

		table.Find("tbody tr").Each(func(j int, row *goquery.Selection) {
			cells := row.Find("td")
			n := cells.Length()
			if n < 2 {
				return // banner/footnote row (colspan collapses it to one <td>)
			}

			label := strings.TrimSpace(cells.Eq(0).Text())
			rateColCount := n - 1

			for ci := 1; ci < n; ci++ {
				rate, err := ratetext.ParseFlatRate(cells.Eq(ci).Text())
				if err != nil {
					continue // floating formula, range, dash, or free text
				}

				tenure := ""
				if rateColCount > 1 {
					hi := len(headers) - (n - ci)
					if hi >= 0 && hi < len(headers) {
						tenure = headers[hi]
					}
				}

				rates = append(rates, models.LoanRate{
					LoanCategory: productName,
					LoanProduct:  productName,
					RateLabel:    label,
					Tenure:       tenure,
					InterestRate: rate,
					SourceURL:    ratesTariffURL,
					ScrapedAt:    scrapeAt,
				})
			}
		})
	})

	if len(rates) == 0 {
		detail := "no loan rows found"
		if len(rowErrs) > 0 {
			detail = strings.Join(rowErrs, "; ")
		}
		return nil, fmt.Errorf("combank: no loan rates parsed from rates-tariff page (selectors likely stale): %s", detail)
	}

	return rates, nil
}
