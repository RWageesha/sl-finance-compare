INSERT INTO banks (name, code)
VALUES
    ('Hatton National Bank', 'HNB'),
    ('Commercial Bank of Ceylon', 'COMB'),
    ('Bank of Ceylon', 'BOC')
ON CONFLICT (code) DO NOTHING;
