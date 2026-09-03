"""Maps each bank scraper's raw output onto the shared products/
product_rates schema — resolving or creating the right product_categories
and products rows along the way. Scraper parsing logic stays untouched;
this is purely the translation step between "raw scraped data" and "what
goes in the database". Port of internal/normalize/*.go.
"""

from db import DB, ProductRate

# --- Fixed deposits -------------------------------------------------------

# Direct rename of the Go RateType enum onto normalized category codes.
_FD_CATEGORY_BY_RATE_TYPE = {
    "normal": "STANDARD_FD",
    "senior": "SENIOR_CITIZEN_FD",
    "special": "SPECIAL_FD",
    "sathkara": "SATHKARA_FD",
    "digital": "DIGITAL_FD",
}


def _title_from_code(code: str) -> str:
    """Turns a category code like "SENIOR_CITIZEN_FD" into a readable
    fallback name ("Senior Citizen Fd"), used only when a category wasn't
    already seeded with a curated name by migration 004.
    """
    return " ".join(w.capitalize() for w in code.lower().split("_"))


def fixed_deposit(db: DB, bank_id: int, bank_name: str, rate: dict) -> ProductRate:
    """Normalizes one scraped fixed deposit rate for bank_name (bank_id)
    into a ProductRate, ready for db.insert_product_rates. `rate` has keys
    matching models.FixedDepositRate: rate_type, tenure_months, min_amount,
    interest_rate, source_url, scraped_at.
    """
    category_code = _FD_CATEGORY_BY_RATE_TYPE.get(rate["rate_type"])
    if category_code is None:
        raise ValueError(f"normalize: unknown fixed deposit rate type {rate['rate_type']!r}")

    category_id = db.get_or_create_category(category_code, _title_from_code(category_code), "FIXED_DEPOSIT")
    product_id = db.get_or_create_product(bank_id, category_id, f"{bank_name} Fixed Deposit")

    return ProductRate(
        product_id=product_id,
        tenure_value=rate["tenure_months"],
        tenure_unit="MONTH",
        min_amount=rate.get("min_amount"),
        interest_rate=rate["interest_rate"],
        source_url=rate.get("source_url", ""),
        scraped_at=rate["scraped_at"],
    )


def fixed_deposit_for(bank_name: str):
    """Returns a (db, bank_id, row) -> ProductRate function bound to
    bank_name, matching the uniform signature scraper/main.py's generic
    normalize-then-validate helper expects (savings/loan don't need the
    bank name, but fixed_deposit does, to build the product name
    "<Bank> Fixed Deposit").
    """

    def _normalize(db: DB, bank_id: int, row: dict) -> ProductRate:
        return fixed_deposit(db, bank_id, bank_name, row)

    return _normalize


# --- Savings ---------------------------------------------------------------

# Keyword found in an account's name -> normalized category code. Checked
# in order (first match wins), same "specific before generic" idiom the
# HNB fixed deposit division classifier already uses.
_SAVINGS_CATEGORY_KEYWORDS = [
    ("senior", "SENIOR_SAVINGS"),
    ("pensioner", "SENIOR_SAVINGS"),
    ("teen", "TEEN_SAVINGS"),
    ("youth", "TEEN_SAVINGS"),
    ("women", "WOMENS_SAVINGS"),
    ("anagi", "WOMENS_SAVINGS"),
    ("kantha", "WOMENS_SAVINGS"),
    ("ladies", "WOMENS_SAVINGS"),
    ("minor", "MINOR_SAVINGS"),
    ("child", "MINOR_SAVINGS"),
    ("singithi", "MINOR_SAVINGS"),
    ("kids", "MINOR_SAVINGS"),
]


def _classify_savings(account_name: str) -> str:
    lower = account_name.lower()
    for keyword, code in _SAVINGS_CATEGORY_KEYWORDS:
        if keyword in lower:
            return code
    return "STANDARD_SAVINGS"


def savings(db: DB, bank_id: int, rate: dict) -> ProductRate:
    """Normalizes one scraped savings account rate for bank_id into a
    ProductRate. `rate` has keys matching models.SavingsRate: account_name,
    balance_tier, interest_rate, source_url, scraped_at.
    """
    category_code = _classify_savings(rate["account_name"])
    category_id = db.get_or_create_category(category_code, _title_from_code(category_code), "SAVINGS")
    product_id = db.get_or_create_product(bank_id, category_id, rate["account_name"])

    return ProductRate(
        product_id=product_id,
        tenure_label=rate.get("balance_tier", ""),
        interest_rate=rate["interest_rate"],
        source_url=rate.get("source_url", ""),
        scraped_at=rate["scraped_at"],
    )


# --- Loans -------------------------------------------------------------

# Keyword found in a loan's category/product name -> normalized category
# code. Checked in order (first match wins) against loan_category, then
# loan_product.
_LOAN_CATEGORY_KEYWORDS = [
    ("home", "HOUSING_LOAN"),
    ("housing", "HOUSING_LOAN"),
    ("personal", "PERSONAL_LOAN"),
    ("education", "EDUCATION_LOAN"),
    ("gold", "GOLD_LOAN"),
    ("pawning", "GOLD_LOAN"),
    ("surekum", "GOLD_LOAN"),
    ("leas", "LEASE"),  # matches "lease"/"leasing"
    ("pensioner", "PENSIONER_LOAN"),
    ("sathkara", "PENSIONER_LOAN"),
]


def _classify_loan(loan_category: str, loan_product: str) -> str:
    for text in (loan_category.lower(), loan_product.lower()):
        for keyword, code in _LOAN_CATEGORY_KEYWORDS:
            if keyword in text:
                return code
    return "OTHER_LOAN"


def loan(db: DB, bank_id: int, rate: dict) -> ProductRate:
    """Normalizes one scraped loan rate for bank_id into a ProductRate.
    `rate` has keys matching models.LoanRate: loan_category, loan_product,
    rate_label, tenure, interest_rate, source_url, scraped_at.
    """
    category_code = _classify_loan(rate["loan_category"], rate["loan_product"])
    category_id = db.get_or_create_category(category_code, _title_from_code(category_code), "LOAN")
    product_id = db.get_or_create_product(bank_id, category_id, rate["loan_product"])

    return ProductRate(
        product_id=product_id,
        tenure_label=rate.get("tenure", ""),
        rate_label=rate.get("rate_label", ""),
        interest_rate=rate["interest_rate"],
        source_url=rate.get("source_url", ""),
        scraped_at=rate["scraped_at"],
    )
