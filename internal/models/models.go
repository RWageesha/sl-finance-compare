package models

import "time"

// Bank represents a financial institution whose products are tracked.
type Bank struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// RateType distinguishes between different fixed deposit rate categories.
type RateType string

const (
	RateTypeNormal   RateType = "normal"
	RateTypeSenior   RateType = "senior"
	RateTypeSpecial  RateType = "special"
	RateTypeSathkara RateType = "sathkara"
	RateTypeDigital  RateType = "digital"
)

// FixedDepositRate represents a single tenure/rate pair scraped from a bank.
type FixedDepositRate struct {
	ID           int64     `json:"id"`
	BankID       int64     `json:"bank_id"`
	BankName     string    `json:"bank_name,omitempty"`
	BankCode     string    `json:"bank_code,omitempty"`
	TenureMonths int       `json:"tenure_months"`
	MinAmount    *float64  `json:"min_amount,omitempty"`
	InterestRate float64   `json:"interest_rate"`
	RateType     RateType  `json:"rate_type"`
	SourceURL    string    `json:"source_url,omitempty"`
	ScrapedAt    time.Time `json:"scraped_at"`
}
