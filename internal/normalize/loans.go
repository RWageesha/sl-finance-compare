package normalize

import (
	"context"
	"strings"

	"github.com/wageesha/sl-finance-compare/internal/db"
	"github.com/wageesha/sl-finance-compare/internal/models"
)

// loanCategoryKeywords maps a keyword found in a loan's category/product
// name to its normalized category code. Checked in order (first match
// wins) against LoanCategory, then LoanProduct.
var loanCategoryKeywords = []struct {
	keyword string
	code    string
}{
	{"home", "HOUSING_LOAN"},
	{"housing", "HOUSING_LOAN"},
	{"personal", "PERSONAL_LOAN"},
	{"education", "EDUCATION_LOAN"},
	{"gold", "GOLD_LOAN"},
	{"pawning", "GOLD_LOAN"},
	{"surekum", "GOLD_LOAN"},
	{"leas", "LEASE"}, // matches "lease"/"leasing"
	{"pensioner", "PENSIONER_LOAN"},
	{"sathkara", "PENSIONER_LOAN"},
}

func classifyLoan(loanCategory, loanProduct string) string {
	for _, text := range []string{strings.ToLower(loanCategory), strings.ToLower(loanProduct)} {
		for _, kw := range loanCategoryKeywords {
			if strings.Contains(text, kw.keyword) {
				return kw.code
			}
		}
	}
	return "OTHER_LOAN"
}

// Loan normalizes one scraped loan rate for bankID into a ProductRate,
// ready for db.InsertProductRates.
func Loan(ctx context.Context, database *db.DB, bankID int64, r models.LoanRate) (models.ProductRate, error) {
	categoryCode := classifyLoan(r.LoanCategory, r.LoanProduct)

	categoryID, err := database.GetOrCreateCategory(ctx, categoryCode, titleFromCode(categoryCode), "LOAN")
	if err != nil {
		return models.ProductRate{}, err
	}

	productID, err := database.GetOrCreateProduct(ctx, bankID, categoryID, r.LoanProduct)
	if err != nil {
		return models.ProductRate{}, err
	}

	return models.ProductRate{
		ProductID:    productID,
		TenureLabel:  r.Tenure,
		RateLabel:    r.RateLabel,
		InterestRate: r.InterestRate,
		SourceURL:    r.SourceURL,
		ScrapedAt:    r.ScrapedAt,
	}, nil
}
