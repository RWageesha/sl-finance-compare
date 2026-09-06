package models

import "time"

// Bank represents a financial institution whose products are tracked.
type Bank struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"`
	Slug     string `json:"slug,omitempty"`
	Status   string `json:"status,omitempty"`    // 'active' | 'inactive' — admin-managed
	BankType string `json:"bank_type,omitempty"` // admin-managed
	Website  string `json:"website,omitempty"`   // admin-managed
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
	// VerificationStatus is admin-review state ('pending' | 'verified' |
	// 'rejected') — it does NOT gate public display: GetLatestRates keeps
	// serving the newest row regardless, so this is a review/audit lane
	// alongside the live site, not a publish gate.
	VerificationStatus string `json:"verification_status,omitempty"`
}

// DataSource identifies one bank/page combination a scraper fetches from,
// so ScrapeRun records have something to attach to.
type DataSource struct {
	ID         int64  `json:"id"`
	BankID     int64  `json:"bank_id"`
	Label      string `json:"label"`
	SourceURL  string `json:"source_url"`
	Status     string `json:"status,omitempty"`      // 'active' | 'disabled'
	SourceType string `json:"source_type,omitempty"` // 'HTML' | 'PDF' | 'API'
}

// ScrapeRun records the outcome of one scrape attempt against a
// DataSource, for basic operational visibility (not a verification/admin
// workflow — just "did the last run against this source succeed").
type ScrapeRun struct {
	ID           int64      `json:"id"`
	SourceID     int64      `json:"source_id"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Status       string     `json:"status"` // success | partial | failed
	RecordsFound int        `json:"records_found"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// AdminUser is a member of the FindRate LK admin team.
type AdminUser struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         string     `json:"role"`   // 'super_admin' | 'admin' | 'editor' | 'viewer'
	Status       string     `json:"status"` // 'active' | 'disabled'
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// AuditLog is one accountability record for an admin action.
type AuditLog struct {
	ID            int64     `json:"id"`
	AdminID       *int64    `json:"admin_id,omitempty"`
	AdminUsername string    `json:"admin_username"`
	Action        string    `json:"action"`
	RecordRef     string    `json:"record_ref"`
	OldValue      string    `json:"old_value,omitempty"`
	NewValue      string    `json:"new_value,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// UserReport is a correction submitted through the public site's Report
// Issue form.
type UserReport struct {
	ID            int64      `json:"id"`
	BankName      string     `json:"bank_name"`
	ProductLabel  string     `json:"product_label"`
	IssueType     string     `json:"issue_type"`
	CurrentValue  string     `json:"current_value,omitempty"`
	CorrectValue  string     `json:"correct_value"`
	SourceURL     string     `json:"source_url"`
	Description   string     `json:"description,omitempty"`
	Status        string     `json:"status"` // 'pending' | 'resolved' | 'rejected'
	CreatedAt     time.Time  `json:"created_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	ResolvedByID  *int64     `json:"resolved_by,omitempty"`
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
