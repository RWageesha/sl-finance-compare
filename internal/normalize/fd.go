package normalize

import (
	"context"
	"fmt"

	"github.com/wageesha/sl-finance-compare/internal/db"
	"github.com/wageesha/sl-finance-compare/internal/models"
)

// fdCategoryByRateType is a direct rename of the existing RateType enum
// onto normalized category codes.
var fdCategoryByRateType = map[models.RateType]string{
	models.RateTypeNormal:   "STANDARD_FD",
	models.RateTypeSenior:   "SENIOR_CITIZEN_FD",
	models.RateTypeSpecial:  "SPECIAL_FD",
	models.RateTypeSathkara: "SATHKARA_FD",
	models.RateTypeDigital:  "DIGITAL_FD",
}

// FixedDeposit normalizes one scraped fixed deposit rate for bankName
// (bankID) into a ProductRate, ready for db.InsertProductRates.
func FixedDeposit(ctx context.Context, database *db.DB, bankID int64, bankName string, r models.FixedDepositRate) (models.ProductRate, error) {
	categoryCode, ok := fdCategoryByRateType[r.RateType]
	if !ok {
		return models.ProductRate{}, fmt.Errorf("normalize: unknown fixed deposit rate type %q", r.RateType)
	}

	categoryID, err := database.GetOrCreateCategory(ctx, categoryCode, titleFromCode(categoryCode), "FIXED_DEPOSIT")
	if err != nil {
		return models.ProductRate{}, err
	}

	productID, err := database.GetOrCreateProduct(ctx, bankID, categoryID, bankName+" Fixed Deposit")
	if err != nil {
		return models.ProductRate{}, err
	}

	tenure := r.TenureMonths
	return models.ProductRate{
		ProductID:    productID,
		TenureValue:  &tenure,
		TenureUnit:   "MONTH",
		MinAmount:    r.MinAmount,
		InterestRate: r.InterestRate,
		SourceURL:    r.SourceURL,
		ScrapedAt:    r.ScrapedAt,
	}, nil
}
