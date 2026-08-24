// Command scraper runs a one-shot pipeline: fetch each configured bank's
// rates page, parse it, and persist the results to Postgres.
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
	"github.com/wageesha/sl-finance-compare/internal/scrapers/boc"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/combank"
	"github.com/wageesha/sl-finance-compare/internal/scrapers/hnb"
)

// bankScraper describes one bank's scrape pipeline so it can be run
// generically: fetch raw bytes, optionally dump them for debugging, parse
// into rate records, then persist.
type bankScraper struct {
	label     string
	bankName  string
	bankCode  string
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
			label:     "hnb",
			bankName:  hnb.BankName,
			bankCode:  hnb.BankCode,
			fetch:     hnb.FetchRatesJSON,
			parse:     hnb.ParseFixedDeposits,
			debugEnv:  "HNB_DEBUG_DUMP_JSON",
			debugFile: "hnb_fixed_deposits.debug.json",
		},
		{
			label:     "combank",
			bankName:  combank.BankName,
			bankCode:  combank.BankCode,
			fetch:     combank.FetchPage,
			parse:     combank.ParseFixedDeposits,
			debugEnv:  "COMBANK_DEBUG_DUMP_HTML",
			debugFile: "combank_fixed_deposits.debug.html",
		},
		{
			label:     "boc",
			bankName:  boc.BankName,
			bankCode:  boc.BankCode,
			fetch:     boc.FetchPage,
			parse:     boc.ParseFixedDeposits,
			debugEnv:  "BOC_DEBUG_DUMP_HTML",
			debugFile: "boc_fixed_deposits.debug.html",
		},
	}

	// One bank's site being down or having changed its markup shouldn't
	// block scraping the others, so run every scraper and only report
	// failure at the end (still surfacing every individual error via log).
	var scrapeErrs []error
	for _, s := range scrapers {
		if err := scrapeBank(ctx, database, s); err != nil {
			log.Printf("%s: scrape failed: %v", s.label, err)
			scrapeErrs = append(scrapeErrs, fmt.Errorf("%s: %w", s.label, err))
		}
	}

	return errors.Join(scrapeErrs...)
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

	rates, parseErr := s.parse(raw)
	if len(rates) == 0 {
		// No usable data at all — this is fatal, there's nothing to insert.
		return parseErr
	}
	if parseErr != nil {
		// Partial parse: log the problem but proceed with what we got,
		// so a few malformed rows don't block an otherwise good scrape.
		log.Printf("%s: warning: %v", s.label, parseErr)
	}

	bankID, err := database.GetOrCreateBank(ctx, s.bankName, s.bankCode)
	if err != nil {
		return err
	}

	for i := range rates {
		rates[i].BankID = bankID
	}

	if err := database.InsertFixedDepositRates(ctx, rates); err != nil {
		return err
	}

	log.Printf("%s: inserted %d fixed deposit rate(s) for %s", s.label, len(rates), s.bankName)
	return nil
}
