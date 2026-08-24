// Package boc scrapes Fixed Deposit interest rates from Bank of Ceylon's
// public rates & tariff page. The page renders server-side with real
// <table> markup, but each cell holds trilingual text (Sinhala/Tamil/
// English) separated by <br> tags, with English always last — so cells
// need a bit more than plain text extraction to isolate the English
// tenure label.
package boc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/wageesha/sl-finance-compare/internal/models"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/ratetext"
)

const (
	BankName = "Bank of Ceylon"
	BankCode = "BOC"

	ratesURL = "https://www.boc.lk/rates-tariff"

	// The rates & tariff page bundles many unrelated sections (exchange
	// rates, loan rates, savings, insurance info, a suspended senior
	// citizen scheme, a scheme terminated in 2009, etc). This is the only
	// section title we want — it's the current, active rupee FD table
	// available for new deposits.
	fixedDepositsSectionTitle = "Rupee Fixed Deposits"

	standardRateCol = 1

	requestTimeout = 15 * time.Second
)

var brSplitRe = regexp.MustCompile(`(?i)<br\s*/?>`)
var tagStripRe = regexp.MustCompile(`<[^>]+>`)

// FetchPage retrieves the raw HTML of BOC's rates & tariff page.
func FetchPage(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ratesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("boc: build request: %w", err)
	}
	// This site's WAF rejects requests without a browser-like UA (403).
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36")
	req.Header.Set("Accept", "text/html")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("boc: fetch %s: %w", ratesURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("boc: unexpected status %d fetching %s: %s", resp.StatusCode, ratesURL, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("boc: read response body: %w", err)
	}

	return body, nil
}

// ParseFixedDeposits extracts tenure/rate pairs from the "Rupee Fixed
// Deposits" section of BOC's rates & tariff page. Like ComBank, tenures
// of 1 year and above get multiple rows per tenure — one per payout
// frequency (monthly/annually/at maturity) — and only the "at maturity"
// variant is kept per tenure so results stay comparable across banks.
// Rows explicitly for "Senior Citizens" are captured separately as
// RateTypeSenior.
func ParseFixedDeposits(html []byte) ([]models.FixedDepositRate, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("boc: parse html: %w", err)
	}

	table := findFixedDepositsTable(doc)
	if table == nil || table.Length() == 0 {
		return nil, fmt.Errorf("boc: %q section/table not found (page structure likely changed)", fixedDepositsSectionTitle)
	}

	var (
		rates    []models.FixedDepositRate
		rowErrs  []string
		scrapeAt = time.Now().UTC()
	)

	table.Find("tbody tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() <= standardRateCol {
			rowErrs = append(rowErrs, fmt.Sprintf("row %d: too few columns", i))
			return
		}

		tenureCellHTML, _ := cells.Eq(0).Html()
		label := lastBrSegmentText(tenureCellHTML)
		if label == "" {
			return // stray/empty row
		}

		lowerLabel := strings.ToLower(label)
		hasVariant := strings.Contains(lowerLabel, "interest paid") || strings.Contains(lowerLabel, "interest at")
		isMaturity := strings.Contains(lowerLabel, "at maturity")
		if hasVariant && !isMaturity {
			return // skip "paid monthly"/"paid annually" rows; keep only maturity payout
		}

		rateType := models.RateTypeNormal
		if strings.Contains(lowerLabel, "senior") {
			rateType = models.RateTypeSenior
		}

		months, err := ratetext.ParseTenureMonths(label)
		if err != nil {
			rowErrs = append(rowErrs, fmt.Sprintf("row %d: tenure %q: %v", i, label, err))
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
			RateType:     rateType,
			SourceURL:    ratesURL,
			ScrapedAt:    scrapeAt,
		})
	})

	if len(rates) == 0 {
		detail := "no rows parsed"
		if len(rowErrs) > 0 {
			detail = strings.Join(rowErrs, "; ")
		}
		return nil, fmt.Errorf("boc: no fixed deposit rates parsed from page (selectors likely stale): %s", detail)
	}

	if len(rowErrs) > 0 {
		return rates, fmt.Errorf("boc: parsed %d rate(s) but skipped %d malformed row(s): %s",
			len(rates), len(rowErrs), strings.Join(rowErrs, "; "))
	}

	return rates, nil
}

// findFixedDepositsTable locates the h3 section titled
// fixedDepositsSectionTitle and returns its rate table. The page's
// repeating structure is: <h3 class="sub-title">Title</h3><h4 .../>
// <div class="exchangerate-table-wrap">...<table class="ck-table-resized">.
func findFixedDepositsTable(doc *goquery.Document) *goquery.Selection {
	var table *goquery.Selection
	doc.Find("h3.sub-title").EachWithBreak(func(i int, h3 *goquery.Selection) bool {
		if strings.TrimSpace(h3.Text()) != fixedDepositsSectionTitle {
			return true // keep looking
		}
		wrap := h3.Next().Next() // h4 -> div.exchangerate-table-wrap
		found := wrap.Find("table.ck-table-resized").First()
		if found.Length() > 0 {
			table = found
		}
		return false // stop, whether or not we found a table
	})
	return table
}

// lastBrSegmentText takes a table cell's inner HTML containing
// <br>-separated multilingual text and returns just the last segment
// (English, by this page's convention), with any remaining tags stripped
// and entities normalized.
func lastBrSegmentText(cellHTML string) string {
	parts := brSplitRe.Split(cellHTML, -1)
	last := parts[len(parts)-1]
	last = tagStripRe.ReplaceAllString(last, "")
	last = strings.ReplaceAll(last, "&nbsp;", " ")
	last = strings.Join(strings.Fields(last), " ")
	return strings.TrimSpace(last)
}

// Scrape fetches the BOC rates page and parses it into rate records. The
// returned rates do not yet have BankID set — the caller is responsible
// for resolving/creating the bank row and setting it before insertion.
func Scrape(ctx context.Context) ([]models.FixedDepositRate, error) {
	body, err := FetchPage(ctx)
	if err != nil {
		return nil, err
	}
	return ParseFixedDeposits(body)
}
