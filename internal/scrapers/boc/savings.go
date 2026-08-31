package boc

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/wageesha/sl-finance-compare/internal/models"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/ratetext"
)

const (
	savingsSectionTitle = "Rupee – Savings Deposits"
	savingsAccountTable = "SAVINGS DEPOSIT ACCOUNT"
)

// paSuffixRe strips a trailing "p.a."/"p.a" unit annotation this table's
// rate cells carry (e.g. "2.00% p.a."), which ratetext.ParseFlatRate
// otherwise correctly rejects as not a bare percentage.
var paSuffixRe = regexp.MustCompile(`(?i)\s*p\.a\.?\s*$`)

// ParseSavings extracts savings account rates from the "Rupee – Savings
// Deposits" section of BOC's rates & tariff page. That section actually
// bundles three tables: a Savings Certificate discount-price table
// (existing customers only, and a completely different shape — skipped by
// only targeting the specific "SAVINGS DEPOSIT ACCOUNT" table by its
// banner text), the flat-rate savings account table this function reads,
// and an "Other Deposit Schemes" table whose rows are both explicitly
// "now suspended" (ratetext.ParseFlatRate rejects that text automatically,
// no special-casing needed — though it's moot since that table isn't the
// one this function looks at anyway).
func ParseSavings(html []byte) ([]models.SavingsRate, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("boc: parse html: %w", err)
	}

	wrap := findSectionWrap(doc, savingsSectionTitle)
	if wrap == nil {
		return nil, fmt.Errorf("boc: %q section not found (page structure likely changed)", savingsSectionTitle)
	}

	var table *goquery.Selection
	wrap.Find("table").EachWithBreak(func(_ int, t *goquery.Selection) bool {
		if strings.Contains(t.Text(), savingsAccountTable) {
			table = t
			return false
		}
		return true
	})
	if table == nil {
		return nil, fmt.Errorf("boc: %q table not found within %q section", savingsAccountTable, savingsSectionTitle)
	}

	var (
		rates    []models.SavingsRate
		rowErrs  []string
		scrapeAt = time.Now().UTC()
	)

	table.Find("tbody tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}

		nameHTML, _ := cells.Eq(0).Html()
		accountName := lastBrSegmentText(nameHTML)
		if accountName == "" {
			return // the table's own banner row, or a stray/empty row
		}

		rateText := paSuffixRe.ReplaceAllString(cells.Eq(1).Text(), "")
		rate, err := ratetext.ParseFlatRate(rateText)
		if err != nil {
			rowErrs = append(rowErrs, fmt.Sprintf("row %d: rate for %q: %v", i, accountName, err))
			return
		}

		rates = append(rates, models.SavingsRate{
			AccountName:  accountName,
			InterestRate: rate,
			SourceURL:    ratesURL,
			ScrapedAt:    scrapeAt,
		})
	})

	if len(rates) == 0 {
		detail := "no savings rows found"
		if len(rowErrs) > 0 {
			detail = strings.Join(rowErrs, "; ")
		}
		return nil, fmt.Errorf("boc: no savings rates parsed from %q table: %s", savingsAccountTable, detail)
	}

	if len(rowErrs) > 0 {
		return rates, fmt.Errorf("boc: parsed %d savings rate(s) but skipped %d malformed row(s): %s",
			len(rates), len(rowErrs), strings.Join(rowErrs, "; "))
	}

	return rates, nil
}

// findSectionWrap locates the h3 titled title and returns the div holding
// its content — the page's repeating structure is
// <h3 class="sub-title">Title</h3><h4 .../><div class="exchangerate-table-wrap">...
// same layout findFixedDepositsTable (fd.go) already relies on for the
// "Rupee Fixed Deposits" section.
func findSectionWrap(doc *goquery.Document, title string) *goquery.Selection {
	var wrap *goquery.Selection
	doc.Find("h3.sub-title").EachWithBreak(func(_ int, h3 *goquery.Selection) bool {
		if strings.TrimSpace(h3.Text()) != title {
			return true
		}
		w := h3.Next().Next()
		if w.Length() > 0 {
			wrap = w
		}
		return false
	})
	return wrap
}
