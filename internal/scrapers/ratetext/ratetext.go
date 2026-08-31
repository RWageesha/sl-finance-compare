// Package ratetext parses the small vocabulary of tenure and interest-rate
// strings that Sri Lankan bank rate tables tend to use (e.g. "12 Months",
// "1 Year", "9.55%"). It's shared across per-bank scraper packages so each
// one doesn't reimplement the same parsing.
package ratetext

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// words maps how banks sometimes spell out non-numeric tenure periods to a
// number of months. Extend this if a bank uses a new label.
var words = map[string]int{
	"one month":     1,
	"three months":  3,
	"six months":    6,
	"twelve months": 12,
}

var tenureNumberRe = regexp.MustCompile(`(\d+)\s*(day|month|year)s?`)
var rateNumberRe = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%?`)
var flatRateRe = regexp.MustCompile(`^\d+(?:\.\d+)?\s*%?$`)

// ParseTenureMonths converts strings like "3 Months", "1 Year", "30 Days",
// or one of the known word forms into a whole number of months. Day-based
// tenures round up to a 1 month floor (0 months is not a valid tenure).
// Extra surrounding text (e.g. "12 Months -Interest at maturity (LKR)") is
// tolerated — only the first number+unit match is used.
func ParseTenureMonths(s string) (int, error) {
	normalized := strings.ToLower(strings.TrimSpace(s))
	if months, ok := words[normalized]; ok {
		return months, nil
	}

	m := tenureNumberRe.FindStringSubmatch(normalized)
	if m == nil {
		return 0, fmt.Errorf("unrecognized tenure format")
	}

	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, fmt.Errorf("invalid tenure number: %w", err)
	}

	switch m[2] {
	case "year":
		return n * 12, nil
	case "month":
		return n, nil
	case "day":
		if n <= 0 {
			return 0, fmt.Errorf("non-positive day tenure")
		}
		months := n / 30
		if months < 1 {
			months = 1
		}
		return months, nil
	default:
		return 0, fmt.Errorf("unrecognized tenure unit %q", m[2])
	}
}

// ParseRate extracts a percentage value like "13.50%", "13.5 %", or a bare
// "13.50" as a float64.
func ParseRate(s string) (float64, error) {
	m := rateNumberRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, fmt.Errorf("unrecognized rate format")
	}
	rate, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid rate number: %w", err)
	}
	if rate <= 0 || rate > 100 {
		return 0, fmt.Errorf("rate %.2f out of plausible range", rate)
	}
	return rate, nil
}

// ParseFlatRate is a stricter sibling of ParseRate: the entire trimmed
// string must be a bare percentage, with no surrounding text. This is used
// to tell a real, comparable rate ("13.50%") apart from a floating-rate
// formula ("AWPLR + 2.50%"), a range ("12.50% - 17.75%"), or a placeholder
// ("-", "Please refer Treasury Division") — none of which reduce to a
// single comparable number.
func ParseFlatRate(s string) (float64, error) {
	trimmed := strings.TrimSpace(s)
	if !flatRateRe.MatchString(trimmed) {
		return 0, fmt.Errorf("not a flat percentage")
	}
	return ParseRate(trimmed)
}
