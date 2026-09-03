"""Plausibility checks a ProductRate must pass before being inserted, as a
boundary separate from the per-cell parsing checks (ratetext.parse_rate/
parse_flat_rate) each scraper already applies while extracting values from
a page. Port of internal/validate/validate.go.
"""

from db import ProductRate


class ValidationError(ValueError):
    pass


def rate(r: ProductRate) -> None:
    """Checks that r is plausible enough to store. Re-checks the interest
    rate bound ratetext.parse_rate already enforces (defense in depth at
    the layer meant to own this), plus the structural fields normalization
    must have filled in. Raises ValidationError if not.
    """
    if not r.product_id:
        raise ValidationError("validate: missing product id")
    if r.interest_rate <= 0 or r.interest_rate > 100:
        raise ValidationError(f"validate: interest rate {r.interest_rate:.3f} out of plausible range")
    if r.tenure_value is not None and r.tenure_value < 0:
        raise ValidationError(f"validate: negative tenure value {r.tenure_value}")
