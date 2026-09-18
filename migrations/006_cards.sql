-- Adds Credit/Debit Card support on top of the existing normalized
-- products/product_rates schema. Cards need two fields no other product
-- type tracks — an annual/joining fee and a minimum income threshold —
-- both nullable since not every scraped card publishes both (debit cards
-- in particular have no minimum income requirement at all). interest_rate
-- stays NOT NULL for every row in this shared table; debit cards (which
-- don't accrue interest) store 0 there, a literal true statement rather
-- than a placeholder, and the frontend never renders an APR field for
-- debit rows regardless.
ALTER TABLE product_rates ADD COLUMN annual_fee NUMERIC(12, 2) CHECK (annual_fee IS NULL OR annual_fee >= 0);
ALTER TABLE product_rates ADD COLUMN min_income NUMERIC(14, 2) CHECK (min_income IS NULL OR min_income >= 0);

INSERT INTO product_categories (code, name, parent_id) VALUES
    ('CARD', 'Cards', NULL)
ON CONFLICT (code) DO NOTHING;

INSERT INTO product_categories (code, name, parent_id)
SELECT code, name, id FROM (
    SELECT id FROM product_categories WHERE code = 'CARD'
) card
CROSS JOIN (VALUES
    ('CREDIT_CARD', 'Credit Card'),
    ('DEBIT_CARD', 'Debit Card')
) AS t(code, name)
ON CONFLICT (code) DO NOTHING;
