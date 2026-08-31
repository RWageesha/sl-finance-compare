package normalize

import (
	"context"
	"strings"

	"github.com/wageesha/sl-finance-compare/internal/db"
	"github.com/wageesha/sl-finance-compare/internal/models"
)

// savingsCategoryKeywords maps a keyword found in an account's name to its
// normalized category code. Checked in order (first match wins), same
// "specific before generic" idiom the HNB fixed deposit division
// classifier already uses.
var savingsCategoryKeywords = []struct {
	keyword string
	code    string
}{
	{"senior", "SENIOR_SAVINGS"},
	{"pensioner", "SENIOR_SAVINGS"},
	{"teen", "TEEN_SAVINGS"},
	{"youth", "TEEN_SAVINGS"},
	{"women", "WOMENS_SAVINGS"},
	{"anagi", "WOMENS_SAVINGS"},
	{"kantha", "WOMENS_SAVINGS"},
	{"ladies", "WOMENS_SAVINGS"},
	{"minor", "MINOR_SAVINGS"},
	{"child", "MINOR_SAVINGS"},
	{"singithi", "MINOR_SAVINGS"},
	{"kids", "MINOR_SAVINGS"},
}

func classifySavings(accountName string) string {
	lower := strings.ToLower(accountName)
	for _, kw := range savingsCategoryKeywords {
		if strings.Contains(lower, kw.keyword) {
			return kw.code
		}
	}
	return "STANDARD_SAVINGS"
}

// Savings normalizes one scraped savings account rate for bankID into a
// ProductRate, ready for db.InsertProductRates.
func Savings(ctx context.Context, database *db.DB, bankID int64, r models.SavingsRate) (models.ProductRate, error) {
	categoryCode := classifySavings(r.AccountName)

	categoryID, err := database.GetOrCreateCategory(ctx, categoryCode, titleFromCode(categoryCode), "SAVINGS")
	if err != nil {
		return models.ProductRate{}, err
	}

	productID, err := database.GetOrCreateProduct(ctx, bankID, categoryID, r.AccountName)
	if err != nil {
		return models.ProductRate{}, err
	}

	return models.ProductRate{
		ProductID:    productID,
		TenureLabel:  r.BalanceTier,
		InterestRate: r.InterestRate,
		SourceURL:    r.SourceURL,
		ScrapedAt:    r.ScrapedAt,
	}, nil
}
