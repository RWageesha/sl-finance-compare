// Package hnb scrapes Fixed Deposit interest rates from Hatton National
// Bank's public rates JSON API.
//
// HNB's rates page (https://www.hnb.lk/fixed-deposits-interest-rates) is a
// client-side rendered React app — the server returns an empty HTML shell,
// and the actual rate tables are fetched by the browser from a backend API
// and rendered with JavaScript. That means an HTML/CSS-selector scraper can
// never see the data: there is no table markup in the server response at
// all. Instead, this package talks to the same JSON API the page itself
// calls.
package hnb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wageesha/sl-finance-compare/internal/models"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/ratetext"
)

const (
	BankName = "Hatton National Bank"
	BankCode = "HNB"

	ratesAPIURL = "https://venus.hnb.lk/api/get_interest_rates_contents"

	fixedDepositsSubCategory = "Fixed Deposits"

	requestTimeout = 15 * time.Second
)

// apiResponse mirrors the shape of https://venus.hnb.lk/api/get_interest_rates_contents.
type apiResponse struct {
	Status int           `json:"status"`
	Msg    string        `json:"msg"`
	Data   []apiCategory `json:"data"`
}

type apiCategory struct {
	Name                    string           `json:"name"`
	InterestRateSubCategory []apiSubCategory `json:"interest_rate_sub_category"`
}

type apiSubCategory struct {
	Name                        string        `json:"name"`
	SubCategoryDivisionApproved []apiDivision `json:"sub_category_division_approved"`
}

type apiDivision struct {
	Title             string         `json:"title"`
	Description       string         `json:"description"`
	TableDataApproved []apiTableData `json:"table_data_approved"`
}

// apiTableData's Data field is itself a JSON-encoded string (not a nested
// object) containing the actual column/row table — the API double-encodes
// it. It must be unmarshalled twice.
type apiTableData struct {
	Data string `json:"data"`
}

type apiTablePayload struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

// FetchRatesJSON retrieves the raw JSON body from HNB's interest rates API.
func FetchRatesJSON(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ratesAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("hnb: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FindRateLKBot/1.0; +https://github.com/)")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hnb: fetch %s: %w", ratesAPIURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("hnb: unexpected status %d fetching %s: %s", resp.StatusCode, ratesAPIURL, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("hnb: read response body: %w", err)
	}

	return body, nil
}

// ParseFixedDeposits extracts tenure/rate pairs from the raw JSON body
// returned by HNB's interest rates API. It looks for the "Fixed Deposits"
// sub-category and reads each of its rate tables (general, special,
// senior citizen, Sathkara, etc.), picking the "Maturity" column as the
// canonical comparable rate per tenure (falling back to "Rate" for tables
// that only publish a single rate, which is paid at maturity by
// definition for those products).
func ParseFixedDeposits(jsonBody []byte) ([]models.FixedDepositRate, error) {
	var resp apiResponse
	if err := json.Unmarshal(jsonBody, &resp); err != nil {
		return nil, fmt.Errorf("hnb: decode api response: %w", err)
	}

	fdSub := findSubCategory(resp, fixedDepositsSubCategory)
	if fdSub == nil {
		return nil, fmt.Errorf("hnb: %q sub-category not found in api response", fixedDepositsSubCategory)
	}

	var (
		rates    []models.FixedDepositRate
		rowErrs  []string
		scrapeAt = time.Now().UTC()
	)

	for _, div := range fdSub.SubCategoryDivisionApproved {
		rateType := classifyRateType(div.Title)

		for _, td := range div.TableDataApproved {
			var payload apiTablePayload
			if err := json.Unmarshal([]byte(td.Data), &payload); err != nil {
				rowErrs = append(rowErrs, fmt.Sprintf("division %q: decode table data: %v", div.Title, err))
				continue
			}

			rateColIdx := findColumnIndex(payload.Columns, "maturity")
			if rateColIdx == -1 {
				rateColIdx = findColumnIndex(payload.Columns, "rate")
			}
			if rateColIdx == -1 {
				rowErrs = append(rowErrs, fmt.Sprintf("division %q: no maturity/rate column in %v", div.Title, payload.Columns))
				continue
			}

			for i, row := range payload.Rows {
				if len(row) <= rateColIdx {
					rowErrs = append(rowErrs, fmt.Sprintf("division %q row %d: too few columns", div.Title, i))
					continue
				}

				tenureText := strings.TrimSpace(row[0])
				rateText := strings.TrimSpace(row[rateColIdx])
				if rateText == "" || rateText == "-" {
					continue // this payout frequency isn't offered for this tenure
				}

				months, err := ratetext.ParseTenureMonths(tenureText)
				if err != nil {
					rowErrs = append(rowErrs, fmt.Sprintf("division %q row %d: tenure %q: %v", div.Title, i, tenureText, err))
					continue
				}

				rate, err := ratetext.ParseRate(rateText)
				if err != nil {
					rowErrs = append(rowErrs, fmt.Sprintf("division %q row %d: rate %q: %v", div.Title, i, rateText, err))
					continue
				}

				rates = append(rates, models.FixedDepositRate{
					TenureMonths: months,
					InterestRate: rate,
					RateType:     rateType,
					SourceURL:    ratesAPIURL,
					ScrapedAt:    scrapeAt,
				})
			}
		}
	}

	if len(rates) == 0 {
		detail := "no fixed deposit rows found"
		if len(rowErrs) > 0 {
			detail = strings.Join(rowErrs, "; ")
		}
		return nil, fmt.Errorf("hnb: no fixed deposit rates parsed from api response (schema likely changed): %s", detail)
	}

	if len(rowErrs) > 0 {
		return rates, fmt.Errorf("hnb: parsed %d rate(s) but skipped %d malformed row(s)/table(s): %s",
			len(rates), len(rowErrs), strings.Join(rowErrs, "; "))
	}

	return rates, nil
}

// findSubCategory searches every category in the response for a
// sub-category with the given name (case-insensitive), regardless of which
// top-level category it lives under — HNB's site nests "Fixed Deposits" and
// the savings sub-categories under the "Savings" category, for instance.
func findSubCategory(resp apiResponse, name string) *apiSubCategory {
	for _, cat := range resp.Data {
		for i := range cat.InterestRateSubCategory {
			sc := &cat.InterestRateSubCategory[i]
			if strings.EqualFold(strings.TrimSpace(sc.Name), name) {
				return sc
			}
		}
	}
	return nil
}

// classifyRateType maps a table's title to a RateType based on keywords.
// Order matters: check specific keywords before falling back to normal,
// since e.g. "Senior Citizen Fixed Deposits" also contains "Fixed Deposits".
func classifyRateType(title string) models.RateType {
	t := strings.ToLower(title)
	switch {
	case strings.Contains(t, "senior"):
		return models.RateTypeSenior
	case strings.Contains(t, "sathkara"):
		return models.RateTypeSathkara
	case strings.Contains(t, "special"):
		return models.RateTypeSpecial
	default:
		return models.RateTypeNormal
	}
}

func findColumnIndex(columns []string, name string) int {
	for i, c := range columns {
		if strings.EqualFold(strings.TrimSpace(c), name) {
			return i
		}
	}
	return -1
}

// Scrape fetches HNB's rates API and parses it into rate records. The
// returned rates do not yet have BankID set — the caller is responsible
// for resolving/creating the bank row and setting it before insertion.
func Scrape(ctx context.Context) ([]models.FixedDepositRate, error) {
	body, err := FetchRatesJSON(ctx)
	if err != nil {
		return nil, err
	}
	return ParseFixedDeposits(body)
}
