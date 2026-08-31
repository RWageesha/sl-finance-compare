package models

import "time"

// Bank represents a financial institution whose products are tracked.
type Bank struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
	Slug string `json:"slug,omitempty"`
}

// ProductCategory is a node in the normalized product taxonomy (e.g.
// "FIXED_DEPOSIT" under parent "DEPOSIT", or "HOUSING_LOAN" under "LOAN").
// Every bank's marketed product name maps onto one of these codes so
// cross-bank comparison can filter/group on a shared vocabulary instead of
// each bank's own naming.
type ProductCategory struct {
	ID       int64  `json:"id"`
	ParentID *int64 `json:"parent_id,omitempty"`
	Code     string `json:"code"`
	Name     string `json:"name"`
}

// Product is one bank's specific offering within a category (e.g. "HNB
// Savings +", "BOC Personal Loan Scheme").
type Product struct {
	ID         int64  `json:"id"`
	BankID     int64  `json:"bank_id"`
	CategoryID int64  `json:"category_id"`
	Name       string `json:"name"`
	Currency   string `json:"currency"`
}

// ProductRate is a single dated rate record for a product. It's the
// normalized landing place for every bank's FixedDepositRate/SavingsRate/
// LoanRate output (see internal/normalize) — tenure_value/tenure_unit
// carry a numeric tenure when one exists (fixed deposits), tenure_label
// carries free text when it doesn't reduce to a single number (loan
// ranges, savings balance tiers), and rate_label disambiguates multiple
// rates for the same product/tenure (customer segment, grace period,
// rate-column name, etc).
type ProductRate struct {
	ID           int64     `json:"id"`
	ProductID    int64     `json:"product_id"`
	BankName     string    `json:"bank_name,omitempty"`
	BankCode     string    `json:"bank_code,omitempty"`
	CategoryCode string    `json:"category_code,omitempty"`
	ProductName  string    `json:"product_name,omitempty"`
	TenureValue  *int      `json:"tenure_value,omitempty"`
	TenureUnit   string    `json:"tenure_unit,omitempty"`
	TenureLabel  string    `json:"tenure_label,omitempty"`
	RateLabel    string    `json:"rate_label,omitempty"`
	MinAmount    *float64  `json:"min_amount,omitempty"`
	InterestRate float64   `json:"interest_rate"`
	SourceURL    string    `json:"source_url,omitempty"`
	Confidence   string    `json:"confidence,omitempty"`
	ScrapedAt    time.Time `json:"scraped_at"`
}

// DataSource identifies one bank/page combination a scraper fetches from,
// so ScrapeRun records have something to attach to.
type DataSource struct {
	ID        int64  `json:"id"`
	BankID    int64  `json:"bank_id"`
	Label     string `json:"label"`
	SourceURL string `json:"source_url"`
}

// ScrapeRun records the outcome of one scrape attempt against a
// DataSource, for basic operational visibility (not a verification/admin
// workflow — just "did the last run against this source succeed").
type ScrapeRun struct {
	ID           int64
	SourceID     int64
	StartedAt    time.Time
	CompletedAt  *time.Time
	Status       string // success | partial | failed
	RecordsFound int
	ErrorMessage string
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

// SavingsRate represents a single savings account rate tier scraped from a
// bank. AccountName identifies the product (e.g. "HNB Savings +",
// "Ordinary Savings Account"); BalanceTier is free text describing the
// balance range the rate applies to, empty when the account has a single
// flat rate.
type SavingsRate struct {
	ID           int64     `json:"id"`
	BankID       int64     `json:"bank_id"`
	BankName     string    `json:"bank_name,omitempty"`
	BankCode     string    `json:"bank_code,omitempty"`
	AccountName  string    `json:"account_name"`
	BalanceTier  string    `json:"balance_tier,omitempty"`
	InterestRate float64   `json:"interest_rate"`
	SourceURL    string    `json:"source_url,omitempty"`
	ScrapedAt    time.Time `json:"scraped_at"`
}

// LoanRate represents a single loan rate scraped from a bank. LoanCategory
// is the bank's grouping (e.g. "Personal Loan"), LoanProduct is the
// specific product within it (e.g. "Loan Against Property (LAP)"). Tenure
// is kept as free text since bank-published loan tenures are often ranges
// ("4 - 5 Years") rather than a single value. RateLabel distinguishes
// multiple rates published for the same product/tenure (e.g. different
// customer segments or a grace-period variant).
type LoanRate struct {
	ID           int64     `json:"id"`
	BankID       int64     `json:"bank_id"`
	BankName     string    `json:"bank_name,omitempty"`
	BankCode     string    `json:"bank_code,omitempty"`
	LoanCategory string    `json:"loan_category"`
	LoanProduct  string    `json:"loan_product"`
	RateLabel    string    `json:"rate_label,omitempty"`
	Tenure       string    `json:"tenure,omitempty"`
	InterestRate float64   `json:"interest_rate"`
	SourceURL    string    `json:"source_url,omitempty"`
	ScrapedAt    time.Time `json:"scraped_at"`
}
