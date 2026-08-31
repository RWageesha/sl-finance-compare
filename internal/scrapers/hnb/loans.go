package hnb

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/wageesha/sl-finance-compare/internal/models"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/ratetext"
)

const loansCategory = "Loans"

// loanLabelColumnNames identifies table columns that describe the loan
// (type/tenure) rather than carry a rate value. Every other column is
// treated as a rate candidate.
var loanLabelColumnNames = map[string]bool{
	"type":           true,
	"tenure":         true,
	"tenor":          true,
	"period":         true,
	"period (years)": true,
	"period (year)":  true,
}

// ParseLoans extracts loan rates from the raw JSON body returned by HNB's
// interest rates API. It walks every division under the "Loans" category.
// Real loan tables mix flat percentages with floating-rate formulas
// ("AWPLR + 2.50%"), ranges, and free text ("Please refer Treasury
// Division") — only cells that are a single flat percentage
// (ratetext.ParseFlatRate) are captured; everything else is silently
// skipped since it isn't a single comparable rate. Divisions covering
// foreign-currency loans are skipped entirely so results stay LKR-only,
// matching how the BOC scraper only captures "Rupee Fixed Deposits".
func ParseLoans(jsonBody []byte) ([]models.LoanRate, error) {
	var resp apiResponse
	if err := json.Unmarshal(jsonBody, &resp); err != nil {
		return nil, fmt.Errorf("hnb: decode api response: %w", err)
	}

	var (
		rates    []models.LoanRate
		rowErrs  []string
		scrapeAt = time.Now().UTC()
	)

	for _, cat := range resp.Data {
		if !strings.EqualFold(strings.TrimSpace(cat.Name), loansCategory) {
			continue
		}

		for _, sub := range cat.InterestRateSubCategory {
			for _, div := range sub.SubCategoryDivisionApproved {
				title := strings.TrimSpace(div.Title)
				if strings.Contains(strings.ToLower(title), "foreign currency") {
					continue
				}

				for _, td := range div.TableDataApproved {
					var payload apiTablePayload
					if err := json.Unmarshal([]byte(td.Data), &payload); err != nil {
						rowErrs = append(rowErrs, fmt.Sprintf("division %q: decode table data: %v", title, err))
						continue
					}

					labelCols, rateCols := classifyLoanColumns(payload.Columns)

					for i, row := range payload.Rows {
						if len(row) < len(payload.Columns) {
							rowErrs = append(rowErrs, fmt.Sprintf("division %q row %d: too few columns", title, i))
							continue
						}

						var labelParts []string
						for _, ci := range labelCols {
							v := strings.TrimSpace(row[ci])
							if v != "" {
								labelParts = append(labelParts, v)
							}
						}
						tenure := strings.Join(labelParts, " ")

						for _, ci := range rateCols {
							rate, err := ratetext.ParseFlatRate(row[ci])
							if err != nil {
								continue // floating formula, range, dash, or free text
							}

							rates = append(rates, models.LoanRate{
								LoanCategory: strings.TrimSpace(sub.Name),
								LoanProduct:  title,
								RateLabel:    strings.TrimSpace(payload.Columns[ci]),
								Tenure:       tenure,
								InterestRate: rate,
								SourceURL:    ratesAPIURL,
								ScrapedAt:    scrapeAt,
							})
						}
					}
				}
			}
		}
	}

	if len(rates) == 0 {
		detail := "no loan rows found"
		if len(rowErrs) > 0 {
			detail = strings.Join(rowErrs, "; ")
		}
		return nil, fmt.Errorf("hnb: no loan rates parsed from api response (schema likely changed): %s", detail)
	}

	if len(rowErrs) > 0 {
		return rates, fmt.Errorf("hnb: parsed %d loan rate(s) but skipped %d malformed row(s): %s",
			len(rates), len(rowErrs), strings.Join(rowErrs, "; "))
	}

	return rates, nil
}

// classifyLoanColumns splits a table's columns into label columns (type,
// tenure/period) and rate-candidate columns (everything else), preserving
// column order.
func classifyLoanColumns(columns []string) (labelCols, rateCols []int) {
	for i, c := range columns {
		key := strings.ToLower(strings.TrimSpace(c))
		if loanLabelColumnNames[key] {
			labelCols = append(labelCols, i)
		} else {
			rateCols = append(rateCols, i)
		}
	}
	return labelCols, rateCols
}
