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

// Loan sections on BOC's rates & tariff page vary wildly in shape, and not
// every one of them is a clean, comparable, currently-open rate:
//   - Personal Loans, Housing Loans, Ran Surekum Naya Seva (pawning),
//     Leasing, and Government Pensioners' Loan Scheme are included below.
//   - Education Loans is skipped: a single product whose rate cell embeds
//     a trilingual label *and* two tenure/rate pairs in a one-off nested
//     shape not worth a bespoke parser for one product.
//   - Development Loans is skipped: an MSME/agriculture grab-bag of
//     ~15 differently-shaped sub-tables (same call as skipping HNB's SME
//     division elsewhere in this codebase).
//   - Advances Against the Deposit is skipped: its rates are a margin
//     over AER/Repo, not an absolute rate — not comparable, same
//     principle as skipping AWPLR-formula cells everywhere else.
//   - Credit Cards is skipped: fees/APR, not an interest rate to compare.
const (
	personalLoansTitle            = "Personal Loans"
	housingLoansTitle             = "Housing Loans"
	ranSurekumNayaSevaTitle       = "Ran Surekum Naya Seva"
	leasingTitle                  = "Leasing"
	governmentPensionersLoanTitle = "Government Pensioners’ Loan Scheme"
)

// englishBrSegment returns the run of <br>-separated segments in cellHTML
// that are plain ASCII text, joined back together. Most tables on this
// page order their trilingual cells Sinhala/Tamil/English (English last —
// see lastBrSegmentText in fd.go), but the Housing Loans table orders its
// Type-of-loan and Period cells English/Sinhala/Tamil (English first) —
// and a couple of its English labels are themselves wrapped across two
// consecutive <br> segments ("Aggregate Housing Loan amount from" /
// "Rs. 5.0 Mn up to Rs. 7.5 Mn"), so a single segment isn't always enough.
// Rather than hardcode a position, this collects whichever contiguous run
// of segments actually looks like English, wherever it falls.
func englishBrSegment(cellHTML string) string {
	var run []string
	for _, seg := range brSplitRe.Split(cellHTML, -1) {
		text := cleanBrSegment(seg)
		if text == "" {
			continue
		}
		if isASCIIText(text) {
			run = append(run, text)
			continue
		}
		if len(run) > 0 {
			break // a non-ASCII segment ends the English run
		}
	}
	if len(run) > 0 {
		return strings.Join(run, " ")
	}
	return lastBrSegmentText(cellHTML) // fallback: single-segment cells
}

func isASCIIText(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

// tenureRateColonRe matches one "<tenure> : <rate>%" segment, e.g.
// "Upto 5 Years : 14.00%" or "Above 7 years to 10 years: 15.50%".
var tenureRateColonRe = regexp.MustCompile(`^(.*?)\s*:\s*(\d+(?:\.\d+)?)\s*%\s*$`)

// tenureRate is one (tenure, rate-text) pair extracted from a cell.
type tenureRate struct {
	tenure string
	rate   string
}

// parseBrSeparatedRatePairs splits a cell's inner HTML on <br>, and reads
// each resulting segment either as a "<tenure> : <rate>%" pair (BOC's
// Personal Loan cells pack several tenure brackets into one cell this
// way) or, when a segment has no such separator, as a single bare rate
// value with no tenure (covers simpler cells that are just "15.00%" with
// no <br> at all, e.g. Ran Surekum Naya Seva).
func parseBrSeparatedRatePairs(cellHTML string) []tenureRate {
	var out []tenureRate
	for _, seg := range brSplitRe.Split(cellHTML, -1) {
		text := cleanBrSegment(seg)
		if text == "" {
			continue
		}
		if m := tenureRateColonRe.FindStringSubmatch(text); m != nil {
			out = append(out, tenureRate{tenure: strings.TrimSpace(m[1]), rate: m[2] + "%"})
		} else {
			out = append(out, tenureRate{rate: text})
		}
	}
	return out
}

// ParseLoans extracts loan rates from the sections of BOC's rates & tariff
// page listed above.
func ParseLoans(html []byte) ([]models.LoanRate, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("boc: parse html: %w", err)
	}

	scrapeAt := time.Now().UTC()

	var rates []models.LoanRate
	rates = append(rates, parsePersonalLoans(doc, scrapeAt)...)
	rates = append(rates, parseHousingLoans(doc, scrapeAt)...)
	rates = append(rates, parseRanSurekumNayaSeva(doc, scrapeAt)...)
	rates = append(rates, parseLeasing(doc, scrapeAt)...)
	rates = append(rates, parseGovernmentPensionersLoanScheme(doc, scrapeAt)...)

	if len(rates) == 0 {
		return nil, fmt.Errorf("boc: no loan rates parsed from rates-tariff page (selectors likely stale)")
	}
	return rates, nil
}

// parsePersonalLoans handles the "Personal Loans" section: two products
// (BOC Personal Loan Scheme, BOC Special Personal Loan Scheme), each a
// single row whose rate cell packs several tenure:rate segments together.
func parsePersonalLoans(doc *goquery.Document, scrapeAt time.Time) []models.LoanRate {
	wrap := findSectionWrap(doc, personalLoansTitle)
	if wrap == nil {
		return nil
	}

	var rates []models.LoanRate
	wrap.Find("table").First().Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}

		nameHTML, _ := cells.Eq(0).Html()
		productName := lastBrSegmentText(nameHTML)
		if productName == "" {
			return
		}

		rateHTML, _ := cells.Eq(1).Html()
		for _, pair := range parseBrSeparatedRatePairs(rateHTML) {
			rate, err := ratetext.ParseFlatRate(pair.rate)
			if err != nil {
				continue
			}
			rates = append(rates, models.LoanRate{
				LoanCategory: personalLoansTitle,
				LoanProduct:  productName,
				RateLabel:    "Fixed Rate",
				Tenure:       pair.tenure,
				InterestRate: rate,
				SourceURL:    ratesURL,
				ScrapedAt:    scrapeAt,
			})
		}
	})
	return rates
}

// parseHousingLoans handles the "Housing Loans" section's rowspan-heavy
// table: S/N and Type-of-loan cells span 2-3 rows, so only the first row
// of each group has 4 <td>s — continuation rows have just 2 (repayment
// period, rate) and reuse the last-seen type. Single-<td> rows are
// section banners (colspan) and are skipped.
func parseHousingLoans(doc *goquery.Document, scrapeAt time.Time) []models.LoanRate {
	wrap := findSectionWrap(doc, housingLoansTitle)
	if wrap == nil {
		return nil
	}

	var (
		rates       []models.LoanRate
		currentType string
	)

	wrap.Find("table").First().Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		var periodCell, rateCell *goquery.Selection

		switch cells.Length() {
		case 4:
			typeHTML, _ := cells.Eq(1).Html()
			currentType = englishBrSegment(typeHTML)
			c := cells.Eq(2)
			periodCell, rateCell = c, cells.Eq(3)
		case 2:
			if currentType == "" {
				return
			}
			c := cells.Eq(0)
			periodCell, rateCell = c, cells.Eq(1)
		default:
			return // banner row (colspan) or unexpected shape
		}

		periodHTML, _ := periodCell.Html()
		period := englishBrSegment(periodHTML)

		rate, err := ratetext.ParseFlatRate(rateCell.Text())
		if err != nil {
			return
		}

		rates = append(rates, models.LoanRate{
			LoanCategory: housingLoansTitle,
			LoanProduct:  currentType,
			RateLabel:    "Fixed Rate",
			Tenure:       period,
			InterestRate: rate,
			SourceURL:    ratesURL,
			ScrapedAt:    scrapeAt,
		})
	})
	return rates
}

// parseRanSurekumNayaSeva handles the pawning/gold loan section: one
// product, one flat rate.
func parseRanSurekumNayaSeva(doc *goquery.Document, scrapeAt time.Time) []models.LoanRate {
	wrap := findSectionWrap(doc, ranSurekumNayaSevaTitle)
	if wrap == nil {
		return nil
	}

	var rates []models.LoanRate
	wrap.Find("table").First().Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}

		nameHTML, _ := cells.Eq(0).Html()
		productName := lastBrSegmentText(nameHTML)
		if productName == "" {
			return // the table's own banner row
		}

		rate, err := ratetext.ParseFlatRate(cells.Eq(1).Text())
		if err != nil {
			return
		}

		rates = append(rates, models.LoanRate{
			LoanCategory: ranSurekumNayaSevaTitle,
			LoanProduct:  productName,
			RateLabel:    "Interest Rate",
			InterestRate: rate,
			SourceURL:    ratesURL,
			ScrapedAt:    scrapeAt,
		})
	})
	return rates
}

// parseLeasing handles the "Leasing" section: three vehicle/machinery
// types, each with a min and max rate column.
func parseLeasing(doc *goquery.Document, scrapeAt time.Time) []models.LoanRate {
	wrap := findSectionWrap(doc, leasingTitle)
	if wrap == nil {
		return nil
	}

	var rates []models.LoanRate
	wrap.Find("table").First().Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 3 {
			return
		}

		nameHTML, _ := cells.Eq(0).Html()
		typeName := lastBrSegmentText(nameHTML)
		if typeName == "" {
			return
		}

		for idx, label := range []string{"Min Rate", "Max Rate"} {
			rate, err := ratetext.ParseFlatRate(cells.Eq(idx + 1).Text())
			if err != nil {
				continue
			}
			rates = append(rates, models.LoanRate{
				LoanCategory: leasingTitle,
				LoanProduct:  typeName,
				RateLabel:    label,
				InterestRate: rate,
				SourceURL:    ratesURL,
				ScrapedAt:    scrapeAt,
			})
		}
	})
	return rates
}

// parseGovernmentPensionersLoanScheme handles its section: one product,
// three tenure brackets each with its own flat rate.
func parseGovernmentPensionersLoanScheme(doc *goquery.Document, scrapeAt time.Time) []models.LoanRate {
	wrap := findSectionWrap(doc, governmentPensionersLoanTitle)
	if wrap == nil {
		return nil
	}

	var rates []models.LoanRate
	wrap.Find("table").First().Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}

		periodHTML, _ := cells.Eq(0).Html()
		period := lastBrSegmentText(periodHTML)
		if period == "" {
			return // the table's own banner row
		}

		rate, err := ratetext.ParseFlatRate(cells.Eq(1).Text())
		if err != nil {
			return
		}

		rates = append(rates, models.LoanRate{
			LoanCategory: governmentPensionersLoanTitle,
			LoanProduct:  governmentPensionersLoanTitle,
			RateLabel:    "Fixed Rate",
			Tenure:       period,
			InterestRate: rate,
			SourceURL:    ratesURL,
			ScrapedAt:    scrapeAt,
		})
	})
	return rates
}
