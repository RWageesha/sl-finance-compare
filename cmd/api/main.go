// Command api serves the REST API for comparing bank financial products.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.Handle("/", http.FileServer(http.Dir("web")))

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

func handleFixedDeposits(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		rates, err := database.GetLatestFixedDeposits(ctx)
		if err != nil {
			log.Printf("api: get latest fixed deposits: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to load fixed deposit rates")
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
