// Package validate holds the plausibility checks a ProductRate must pass
// before being inserted, as a boundary separate from the per-cell parsing
// checks (ratetext.ParseRate/ParseFlatRate) each scraper already applies
// while extracting values from a page.
package validate

import (
	"fmt"

	"github.com/wageesha/sl-finance-compare/internal/models"
)

// Rate checks that r is plausible enough to store. It re-checks the
// interest rate bound ratetext.ParseRate already enforces (defense in
// depth at the layer meant to own this), plus the structural fields that
// normalization must have filled in.
func Rate(r models.ProductRate) error {
	if r.ProductID == 0 {
		return fmt.Errorf("validate: missing product id")
	}
	if r.InterestRate <= 0 || r.InterestRate > 100 {
		return fmt.Errorf("validate: interest rate %.3f out of plausible range", r.InterestRate)
	}
	if r.TenureValue != nil && *r.TenureValue < 0 {
		return fmt.Errorf("validate: negative tenure value %d", *r.TenureValue)
	}
	return nil
}
