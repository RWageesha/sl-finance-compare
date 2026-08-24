// Package combank scrapes Fixed Deposit interest rates from Commercial
// Bank of Ceylon's public rates page. Unlike HNB, this page is
// server-rendered — the rate table is present directly in the HTML
// response, so this scraper parses it with goquery rather than calling a
// backend API.
package combank

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/wageesha/sl-finance-compare/internal/models"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/ratetext"
)

const (
	BankName = "Commercial Bank of Ceylon"
	BankCode = "COMB"

	ratesURL = "https://www.combank.lk/personal-banking/term-deposits/fixed-deposits"

	// The rate table has three columns beyond tenure: standard nominal
	// rate, Annual Effective Rate, and a bonus "eFD" rate for customers
	// who place the deposit through digital banking. Column indices are
	// relative to <td> position within a row (tenure is index 0).
	standardRateCol = 1
	efdRateCol      = 3

	requestTimeout = 15 * time.Second
)

// FetchPage retrieves the raw HTML of the ComBank fixed deposit rates page.
func FetchPage(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ratesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("combank: build request: %w", err)
	}
	// This site's WAF appears to reject requests that don't look like a
	// real browser (a bot-labelled UA got blocked during development).
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "text/html")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("combank: fetch %s: %w", ratesURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("combank: unexpected status %d fetching %s: %s", resp.StatusCode, ratesURL, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("combank: read response body: %w", err)
	}

	return body, nil
}

// ParseFixedDeposits extracts tenure/rate pairs from the ComBank rates page
// HTML. The page lists 1-6 month tenures as single rows (paid at maturity
// by definition), and 12/24/36/48/60 month tenures as multiple rows — one
// per payout frequency (monthly/annually/at maturity). Only the "at
// maturity" variant is kept per tenure, so results are directly comparable
// with other banks' headline maturity rates. Where the page also lists a
// bonus "eFD" rate for digital banking customers, that's captured as a
// separate RateTypeDigital record for the same tenure.
func ParseFixedDeposits(html []byte) ([]models.FixedDepositRate, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("combank: parse html: %w", err)
	}

	var (
		rates    []models.FixedDepositRate
		rowErrs  []string
		scrapeAt = time.Now().UTC()
	)

	doc.Find("table.with-radius tbody tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return // stray/empty row
		}

		label := strings.TrimSpace(cells.Eq(0).Text())
		if label == "" {
			return
		}

		lowerLabel := strings.ToLower(label)
		hasVariant := strings.Contains(lowerLabel, "interest paid") || strings.Contains(lowerLabel, "interest at")
		isMaturity := strings.Contains(lowerLabel, "at maturity")
		if hasVariant && !isMaturity {
			return // skip "paid monthly"/"paid annually" rows; keep only maturity payout
		}

		months, err := ratetext.ParseTenureMonths(label)
		if err != nil {
			rowErrs = append(rowErrs, fmt.Sprintf("row %d: tenure %q: %v", i, label, err))
			return
		}

		if cells.Length() <= standardRateCol {
			rowErrs = append(rowErrs, fmt.Sprintf("row %d: missing standard rate column", i))
			return
		}
		rateText := strings.TrimSpace(cells.Eq(standardRateCol).Text())
		rate, err := ratetext.ParseRate(rateText)
		if err != nil {
			rowErrs = append(rowErrs, fmt.Sprintf("row %d: rate %q: %v", i, rateText, err))
			return
		}

		rates = append(rates, models.FixedDepositRate{
			TenureMonths: months,
			InterestRate: rate,
			RateType:     models.RateTypeNormal,
			SourceURL:    ratesURL,
			ScrapedAt:    scrapeAt,
		})

		if cells.Length() > efdRateCol {
			efdText := strings.TrimSpace(cells.Eq(efdRateCol).Text())
			if efdRate, err := ratetext.ParseRate(efdText); err == nil {
				rates = append(rates, models.FixedDepositRate{
					TenureMonths: months,
					InterestRate: efdRate,
					RateType:     models.RateTypeDigital,
					SourceURL:    ratesURL,
					ScrapedAt:    scrapeAt,
				})
			}
		}
	})

	if len(rates) == 0 {
		detail := "no rows matched selector \"table.with-radius tbody tr\""
		if len(rowErrs) > 0 {
			detail = strings.Join(rowErrs, "; ")
		}
		return nil, fmt.Errorf("combank: no fixed deposit rates parsed from page (selectors likely stale): %s", detail)
	}

	if len(rowErrs) > 0 {
		return rates, fmt.Errorf("combank: parsed %d rate(s) but skipped %d malformed row(s): %s",
			len(rates), len(rowErrs), strings.Join(rowErrs, "; "))
	}

	return rates, nil
}

// Scrape fetches the ComBank rates page and parses it into rate records.
// The returned rates do not yet have BankID set — the caller is
// responsible for resolving/creating the bank row and setting it before
// insertion.
func Scrape(ctx context.Context) ([]models.FixedDepositRate, error) {
	body, err := FetchPage(ctx)
	if err != nil {
		return nil, err
	}
	return ParseFixedDeposits(body)
}
