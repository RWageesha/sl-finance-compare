// Command api serves the REST API for comparing bank financial products.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/wageesha/sl-finance-compare/internal/db"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("api: %v", err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return err
	}

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		return errors.New("DATABASE_URL is not set (check your .env file)")
	}

	// Render (and similar PaaS providers) injects PORT and requires the
	// app to bind to it; API_PORT remains the override for local dev.
	port := os.Getenv("API_PORT")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := db.New(ctx, connString)
	if err != nil {
		return err
	}
	defer database.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/fixed-deposits", handleFixedDeposits(database))
	mux.HandleFunc("GET /api/v1/savings-rates", handleSavingsRates(database))
	mux.HandleFunc("GET /api/v1/loan-rates", handleLoanRates(database))
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.Handle("/", spaFileServer{root: "web", fallback: "200.html"})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("api: listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Printf("api: shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// spaFileServer serves prerendered static files from root, falling back to
// root/fallback (Nitro's generated 200.html SPA shell) for any path
// http.FileServer would 404 on. Most routes here are fully prerendered
// (every page nuxt generate's crawler could reach via a link), but some —
// e.g. FD product-detail pages, whose exact slugs depend on live scraped
// data that doesn't exist at `nuxt generate` time — have no matching
// prerendered file. Client-side navigation to those already works (Vue
// Router handles it once the app is loaded); this covers a fresh, direct,
// or bookmarked load, so the app shell loads and the client-side router
// resolves the route correctly instead of a bare 404.
type spaFileServer struct {
	root     string
	fallback string
}

func (s spaFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf := &bufferedResponse{header: make(http.Header)}
	http.FileServer(http.Dir(s.root)).ServeHTTP(buf, r)

	if buf.status == http.StatusNotFound {
		http.ServeFile(w, r, filepath.Join(s.root, s.fallback))
		return
	}

	for k, v := range buf.header {
		w.Header()[k] = v
	}
	w.WriteHeader(buf.status)
	w.Write(buf.body.Bytes()) //nolint:errcheck // best-effort write to an already-buffered response
}

// bufferedResponse buffers a handler's entire response so ServeHTTP above
// can inspect the status code before deciding whether to relay it or serve
// the SPA fallback instead. Fine for a small static site's file sizes;
// not meant for large payloads.
type bufferedResponse struct {
	header      http.Header
	status      int
	body        bytes.Buffer
	wroteHeader bool
}

func (b *bufferedResponse) Header() http.Header { return b.header }

func (b *bufferedResponse) WriteHeader(status int) {
	if !b.wroteHeader {
		b.status = status
		b.wroteHeader = true
	}
}

func (b *bufferedResponse) Write(p []byte) (int, error) {
	if !b.wroteHeader {
		b.WriteHeader(http.StatusOK)
	}
	return b.body.Write(p)
}

func handleFixedDeposits(database *db.DB) http.HandlerFunc {
	return handleRatesForGroup(database, "FIXED_DEPOSIT", "fixed deposit rates")
}

func handleSavingsRates(database *db.DB) http.HandlerFunc {
	return handleRatesForGroup(database, "SAVINGS", "savings rates")
}

func handleLoanRates(database *db.DB) http.HandlerFunc {
	return handleRatesForGroup(database, "LOAN", "loan rates")
}

// handleRatesForGroup returns a handler serving the latest rates under a
// top-level category group ("FIXED_DEPOSIT", "SAVINGS", or "LOAN") — the
// three product-type endpoints differ only in which group they query.
func handleRatesForGroup(database *db.DB, categoryGroup, errLabel string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		rates, err := database.GetLatestRates(ctx, categoryGroup)
		if err != nil {
			log.Printf("api: get latest %s: %v", errLabel, err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load "+errLabel)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"count": len(rates),
			"data":  rates,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("api: write json response: %v", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
