// Command scraper runs a one-shot pipeline: fetch each configured bank's
// rates page, parse it, normalize the result onto the shared products/
// product_rates schema, and persist it to Postgres.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"github.com/wageesha/sl-finance-compare/internal/db"
	"github.com/wageesha/sl-finance-compare/internal/models"
	"github.com/wageesha/sl-finance-compare/internal/normalize"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/boc"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/combank"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/hnb"
	"github.com/wageesha/sl-finance-compare/internal/validate"
)

// bankScraper describes one bank's fixed-deposit-only scrape pipeline so
// it can be run generically: fetch raw bytes, optionally dump them for
// debugging, parse into rate records, then persist. Banks with more than
// one product type (currently just HNB) are handled by their own
// dedicated function instead — see scrapeHNB.
type bankScraper struct {
	label     string
	bankName  string
	bankCode  string
	sourceURL string
	fetch     func(ctx context.Context) ([]byte, error)
	parse     func(raw []byte) ([]models.FixedDepositRate, error)
	debugEnv  string
	debugFile string
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("scraper: %v", err)
	}
}

func run() error {
	// It's fine if .env doesn't exist (e.g. in CI/production where env
	// vars are set directly) — only fail on real parse errors.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return err
	}

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		return fmt.Errorf("DATABASE_URL is not set (check your .env file)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	database, err := db.New(ctx, connString)
	if err != nil {
		return err
	}
	defer database.Close()

	scrapers := []bankScraper{
		{
			label:     "combank",
			bankName:  combank.BankName,
			bankCode:  combank.BankCode,
			sourceURL: combank.FixedDepositRatesURL,
			fetch:     combank.FetchPage,
			parse:     combank.ParseFixedDeposits,
			debugEnv:  "COMBANK_DEBUG_DUMP_HTML",
			debugFile: "combank_fixed_deposits.debug.html",
		},
	}

	// One bank's site being down or having changed its markup shouldn't
	// block scraping the others, so run every scraper and only report
	// failure at the end (still surfacing every individual error via log).
	var scrapeErrs []error

	if err := scrapeHNB(ctx, database); err != nil {
		log.Printf("hnb: scrape failed: %v", err)
		scrapeErrs = append(scrapeErrs, fmt.Errorf("hnb: %w", err))
	}

	if err := scrapeComBankSavingsAndLoans(ctx, database); err != nil {
		log.Printf("combank: savings/loans scrape failed: %v", err)
		scrapeErrs = append(scrapeErrs, fmt.Errorf("combank savings/loans: %w", err))
	}

	if err := scrapeBOC(ctx, database); err != nil {
		log.Printf("boc: scrape failed: %v", err)
		scrapeErrs = append(scrapeErrs, fmt.Errorf("boc: %w", err))
	}

	for _, s := range scrapers {
		if err := scrapeBank(ctx, database, s); err != nil {
			log.Printf("%s: scrape failed: %v", s.label, err)
			scrapeErrs = append(scrapeErrs, fmt.Errorf("%s: %w", s.label, err))
		}
	}

	return errors.Join(scrapeErrs...)
}

// runScrape wraps one scrape attempt with data_sources/scrape_runs
// bookkeeping: resolves the source row (creating it on first run), then
// records how the attempt went (row count, status, error) once fn
// returns. fn does the actual fetch/parse/normalize/insert work and
// reports how many rows it inserted.
func runScrape(ctx context.Context, database *db.DB, bankID int64, label, sourceURL string, fn func() (int, error)) error {
	sourceID, err := database.GetOrCreateDataSource(ctx, bankID, label, sourceURL)
	if err != nil {
		return err
	}

	started := time.Now().UTC()
	count, fnErr := fn()
	completed := time.Now().UTC()

	status := "success"
	errMsg := ""
	if fnErr != nil {
		status = "failed"
		if count > 0 {
			status = "partial"
		}
		errMsg = fnErr.Error()
	}

	if recErr := database.RecordScrapeRun(ctx, models.ScrapeRun{
		SourceID:     sourceID,
		StartedAt:    started,
		CompletedAt:  &completed,
		Status:       status,
		RecordsFound: count,
		ErrorMessage: errMsg,
	}); recErr != nil {
		log.Printf("%s: warning: failed to record scrape run: %v", label, recErr)
	}

	return fnErr
}

// scrapeHNB fetches HNB's rates API once and parses the same raw bytes
// three ways (fixed deposits, savings accounts, loans) — HNB's API returns
// all of a bank's published rates in a single response, so fetching it
// separately per product type would just mean redundant HTTP calls against
// the same data. Each product type's parse/insert failure is independent:
// one being fatal (e.g. no fixed deposit rows at all) doesn't block the
// others.
func scrapeHNB(ctx context.Context, database *db.DB) error {
	log.Printf("hnb: fetching rates")

	raw, err := hnb.FetchRatesJSON(ctx)
	if err != nil {
		return err
	}

	if os.Getenv("HNB_DEBUG_DUMP_JSON") == "1" {
		if err := os.WriteFile("hnb_rates.debug.json", raw, 0o644); err != nil {
			log.Printf("hnb: warning: failed to write debug dump: %v", err)
		} else {
			log.Printf("hnb: wrote fetched data to hnb_rates.debug.json")
		}
	}

	bankID, err := database.GetOrCreateBank(ctx, hnb.BankName, hnb.BankCode)
	if err != nil {
		return err
	}

	var productErrs []error

	if err := scrapeHNBFixedDeposits(ctx, database, bankID, raw); err != nil {
		log.Printf("hnb: fixed deposits: %v", err)
		productErrs = append(productErrs, fmt.Errorf("fixed deposits: %w", err))
	}
	if err := scrapeHNBSavings(ctx, database, bankID, raw); err != nil {
		log.Printf("hnb: savings: %v", err)
		productErrs = append(productErrs, fmt.Errorf("savings: %w", err))
	}
	if err := scrapeHNBLoans(ctx, database, bankID, raw); err != nil {
		log.Printf("hnb: loans: %v", err)
		productErrs = append(productErrs, fmt.Errorf("loans: %w", err))
	}

	return errors.Join(productErrs...)
}

func scrapeHNBFixedDeposits(ctx context.Context, database *db.DB, bankID int64, raw []byte) error {
	return runScrape(ctx, database, bankID, "hnb-fixed-deposits", hnbRatesAPIURL, func() (int, error) {
		rates, parseErr := hnb.ParseFixedDeposits(raw)
		if len(rates) == 0 {
			return 0, parseErr
		}
		if parseErr != nil {
			log.Printf("hnb: warning: %v", parseErr)
		}

		productRates := normalizeAll(rates, func(r models.FixedDepositRate) (models.ProductRate, error) {
			return normalize.FixedDeposit(ctx, database, bankID, hnb.BankName, r)
		}, "hnb")

		if err := database.InsertProductRates(ctx, productRates); err != nil {
			return len(productRates), err
		}
		log.Printf("hnb: inserted %d fixed deposit rate(s)", len(productRates))
		return len(productRates), nil
	})
}

func scrapeHNBSavings(ctx context.Context, database *db.DB, bankID int64, raw []byte) error {
	return runScrape(ctx, database, bankID, "hnb-savings", hnbRatesAPIURL, func() (int, error) {
		rates, parseErr := hnb.ParseSavingsAccounts(raw)
		if len(rates) == 0 {
			return 0, parseErr
		}
		if parseErr != nil {
			log.Printf("hnb: warning: %v", parseErr)
		}

		productRates := normalizeAll(rates, func(r models.SavingsRate) (models.ProductRate, error) {
			return normalize.Savings(ctx, database, bankID, r)
		}, "hnb")

		if err := database.InsertProductRates(ctx, productRates); err != nil {
			return len(productRates), err
		}
		log.Printf("hnb: inserted %d savings rate(s)", len(productRates))
		return len(productRates), nil
	})
}

func scrapeHNBLoans(ctx context.Context, database *db.DB, bankID int64, raw []byte) error {
	return runScrape(ctx, database, bankID, "hnb-loans", hnbRatesAPIURL, func() (int, error) {
		rates, parseErr := hnb.ParseLoans(raw)
		if len(rates) == 0 {
			return 0, parseErr
		}
		if parseErr != nil {
			log.Printf("hnb: warning: %v", parseErr)
		}

		productRates := normalizeAll(rates, func(r models.LoanRate) (models.ProductRate, error) {
			return normalize.Loan(ctx, database, bankID, r)
		}, "hnb")

		if err := database.InsertProductRates(ctx, productRates); err != nil {
			return len(productRates), err
		}
		log.Printf("hnb: inserted %d loan rate(s)", len(productRates))
		return len(productRates), nil
	})
}

// hnbRatesAPIURL mirrors the unexported constant of the same value inside
// the hnb package (used only as the data_sources.source_url label here).
const hnbRatesAPIURL = "https://venus.hnb.lk/api/get_interest_rates_contents"

// scrapeComBankSavingsAndLoans fetches ComBank's rates & tariff hub page
// once and parses it two ways (savings, loans) — that page bundles every
// product's rate table under one URL, separate from the dedicated Fixed
// Deposit page scrapeBank already handles for ComBank.
func scrapeComBankSavingsAndLoans(ctx context.Context, database *db.DB) error {
	log.Printf("combank: fetching rates-tariff page")

	raw, err := combank.FetchRatesTariffPage(ctx)
	if err != nil {
		return err
	}

	if os.Getenv("COMBANK_DEBUG_DUMP_RATES_TARIFF_HTML") == "1" {
		if err := os.WriteFile("combank_rates_tariff.debug.html", raw, 0o644); err != nil {
			log.Printf("combank: warning: failed to write debug dump: %v", err)
		} else {
			log.Printf("combank: wrote fetched data to combank_rates_tariff.debug.html")
		}
	}

	bankID, err := database.GetOrCreateBank(ctx, combank.BankName, combank.BankCode)
	if err != nil {
		return err
	}

	var productErrs []error

	if err := scrapeComBankSavings(ctx, database, bankID, raw); err != nil {
		log.Printf("combank: savings: %v", err)
		productErrs = append(productErrs, fmt.Errorf("savings: %w", err))
	}
	if err := scrapeComBankLoans(ctx, database, bankID, raw); err != nil {
		log.Printf("combank: loans: %v", err)
		productErrs = append(productErrs, fmt.Errorf("loans: %w", err))
	}

	return errors.Join(productErrs...)
}

func scrapeComBankSavings(ctx context.Context, database *db.DB, bankID int64, raw []byte) error {
	return runScrape(ctx, database, bankID, "combank-savings", combank.RatesTariffURL, func() (int, error) {
		rates, parseErr := combank.ParseSavings(raw)
		if len(rates) == 0 {
			return 0, parseErr
		}
		if parseErr != nil {
			log.Printf("combank: warning: %v", parseErr)
		}

		productRates := normalizeAll(rates, func(r models.SavingsRate) (models.ProductRate, error) {
			return normalize.Savings(ctx, database, bankID, r)
		}, "combank")

		if err := database.InsertProductRates(ctx, productRates); err != nil {
			return len(productRates), err
		}
		log.Printf("combank: inserted %d savings rate(s)", len(productRates))
		return len(productRates), nil
	})
}

func scrapeComBankLoans(ctx context.Context, database *db.DB, bankID int64, raw []byte) error {
	return runScrape(ctx, database, bankID, "combank-loans", combank.RatesTariffURL, func() (int, error) {
		rates, parseErr := combank.ParseLoans(raw)
		if len(rates) == 0 {
			return 0, parseErr
		}
		if parseErr != nil {
			log.Printf("combank: warning: %v", parseErr)
		}

		productRates := normalizeAll(rates, func(r models.LoanRate) (models.ProductRate, error) {
			return normalize.Loan(ctx, database, bankID, r)
		}, "combank")

		if err := database.InsertProductRates(ctx, productRates); err != nil {
			return len(productRates), err
		}
		log.Printf("combank: inserted %d loan rate(s)", len(productRates))
		return len(productRates), nil
	})
}

// normalizeAll runs normalizeFn over every raw scraped row, validates each
// result, and returns only the rows that passed both steps — logging a
// warning (prefixed with label) for anything dropped, same "skip and log"
// posture the scrapers themselves already use for malformed rows.
func normalizeAll[T any](rows []T, normalizeFn func(T) (models.ProductRate, error), label string) []models.ProductRate {
	out := make([]models.ProductRate, 0, len(rows))
	for _, row := range rows {
		pr, err := normalizeFn(row)
		if err != nil {
			log.Printf("%s: warning: normalize: %v", label, err)
			continue
		}
		if err := validate.Rate(pr); err != nil {
			log.Printf("%s: warning: %v", label, err)
			continue
		}
		out = append(out, pr)
	}
	return out
}

// scrapeBOC fetches BOC's rates & tariff page once and parses it three
// ways (fixed deposits, savings, loans) — same fetch-once-parse-many-times
// structure as scrapeHNB, since BOC (unlike ComBank) publishes all three
// product types on the one page already used for its fixed deposit rates.
func scrapeBOC(ctx context.Context, database *db.DB) error {
	log.Printf("boc: fetching rates-tariff page")

	raw, err := boc.FetchPage(ctx)
	if err != nil {
		return err
	}

	if os.Getenv("BOC_DEBUG_DUMP_HTML") == "1" {
		if err := os.WriteFile("boc_fixed_deposits.debug.html", raw, 0o644); err != nil {
			log.Printf("boc: warning: failed to write debug dump: %v", err)
		} else {
			log.Printf("boc: wrote fetched data to boc_fixed_deposits.debug.html")
		}
	}

	bankID, err := database.GetOrCreateBank(ctx, boc.BankName, boc.BankCode)
	if err != nil {
		return err
	}

	var productErrs []error

	if err := scrapeBOCFixedDeposits(ctx, database, bankID, raw); err != nil {
		log.Printf("boc: fixed deposits: %v", err)
		productErrs = append(productErrs, fmt.Errorf("fixed deposits: %w", err))
	}
	if err := scrapeBOCSavings(ctx, database, bankID, raw); err != nil {
		log.Printf("boc: savings: %v", err)
		productErrs = append(productErrs, fmt.Errorf("savings: %w", err))
	}
	if err := scrapeBOCLoans(ctx, database, bankID, raw); err != nil {
		log.Printf("boc: loans: %v", err)
		productErrs = append(productErrs, fmt.Errorf("loans: %w", err))
	}

	return errors.Join(productErrs...)
}

func scrapeBOCFixedDeposits(ctx context.Context, database *db.DB, bankID int64, raw []byte) error {
	return runScrape(ctx, database, bankID, "boc-fixed-deposits", boc.RatesTariffURL, func() (int, error) {
		rates, parseErr := boc.ParseFixedDeposits(raw)
		if len(rates) == 0 {
			return 0, parseErr
		}
		if parseErr != nil {
			log.Printf("boc: warning: %v", parseErr)
		}

		productRates := normalizeAll(rates, func(r models.FixedDepositRate) (models.ProductRate, error) {
			return normalize.FixedDeposit(ctx, database, bankID, boc.BankName, r)
		}, "boc")

		if err := database.InsertProductRates(ctx, productRates); err != nil {
			return len(productRates), err
		}
		log.Printf("boc: inserted %d fixed deposit rate(s)", len(productRates))
		return len(productRates), nil
	})
}

func scrapeBOCSavings(ctx context.Context, database *db.DB, bankID int64, raw []byte) error {
	return runScrape(ctx, database, bankID, "boc-savings", boc.RatesTariffURL, func() (int, error) {
		rates, parseErr := boc.ParseSavings(raw)
		if len(rates) == 0 {
			return 0, parseErr
		}
		if parseErr != nil {
			log.Printf("boc: warning: %v", parseErr)
		}

		productRates := normalizeAll(rates, func(r models.SavingsRate) (models.ProductRate, error) {
			return normalize.Savings(ctx, database, bankID, r)
		}, "boc")

		if err := database.InsertProductRates(ctx, productRates); err != nil {
			return len(productRates), err
		}
		log.Printf("boc: inserted %d savings rate(s)", len(productRates))
		return len(productRates), nil
	})
}

func scrapeBOCLoans(ctx context.Context, database *db.DB, bankID int64, raw []byte) error {
	return runScrape(ctx, database, bankID, "boc-loans", boc.RatesTariffURL, func() (int, error) {
		rates, parseErr := boc.ParseLoans(raw)
		if len(rates) == 0 {
			return 0, parseErr
		}
		if parseErr != nil {
			log.Printf("boc: warning: %v", parseErr)
		}

		productRates := normalizeAll(rates, func(r models.LoanRate) (models.ProductRate, error) {
			return normalize.Loan(ctx, database, bankID, r)
		}, "boc")

		if err := database.InsertProductRates(ctx, productRates); err != nil {
			return len(productRates), err
		}
		log.Printf("boc: inserted %d loan rate(s)", len(productRates))
		return len(productRates), nil
	})
}

func scrapeBank(ctx context.Context, database *db.DB, s bankScraper) error {
	log.Printf("%s: fetching fixed deposit rates", s.label)

	raw, err := s.fetch(ctx)
	if err != nil {
		return err
	}

	if os.Getenv(s.debugEnv) == "1" {
		if err := os.WriteFile(s.debugFile, raw, 0o644); err != nil {
			log.Printf("%s: warning: failed to write debug dump: %v", s.label, err)
		} else {
			log.Printf("%s: wrote fetched data to %s", s.label, s.debugFile)
		}
	}

	bankID, err := database.GetOrCreateBank(ctx, s.bankName, s.bankCode)
	if err != nil {
		return err
	}

	return runScrape(ctx, database, bankID, s.label+"-fixed-deposits", s.sourceURL, func() (int, error) {
		rates, parseErr := s.parse(raw)
		if len(rates) == 0 {
			// No usable data at all — this is fatal, there's nothing to insert.
			return 0, parseErr
		}
		if parseErr != nil {
			// Partial parse: log the problem but proceed with what we got,
			// so a few malformed rows don't block an otherwise good scrape.
			log.Printf("%s: warning: %v", s.label, parseErr)
		}

		productRates := normalizeAll(rates, func(r models.FixedDepositRate) (models.ProductRate, error) {
			return normalize.FixedDeposit(ctx, database, bankID, s.bankName, r)
		}, s.label)

		if err := database.InsertProductRates(ctx, productRates); err != nil {
			return len(productRates), err
		}
		log.Printf("%s: inserted %d fixed deposit rate(s) for %s", s.label, len(productRates), s.bankName)
		return len(productRates), nil
	})
}
