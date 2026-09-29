package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Media (ad creatives, hero images, bank logos) can't be written to local
// disk — Render's web service has no persistent disk (see
// render.yaml's header comment: the whole frontend build is regenerated
// fresh on every deploy), so anything saved at runtime is lost on the
// next deploy or restart. Supabase Storage is used instead — the project
// already has a Supabase account for Postgres, so this is a new bucket
// in an existing project, not a new vendor. No SDK: Storage's REST API
// is a plain authenticated HTTP PUT, matching this codebase's
// minimal-dependency style.
const defaultStorageBucket = "site-media"

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9.\-]+`)

func handleAdminMediaUpload() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		supabaseURL := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
		serviceKey := os.Getenv("SUPABASE_SERVICE_ROLE_KEY")
		if supabaseURL == "" || serviceKey == "" {
			writeJSONError(w, http.StatusServiceUnavailable, "media storage is not configured (SUPABASE_URL / SUPABASE_SERVICE_ROLE_KEY missing)")
			return
		}
		bucket := os.Getenv("SUPABASE_STORAGE_BUCKET")
		if bucket == "" {
			bucket = defaultStorageBucket
		}

		if err := r.ParseMultipartForm(25 << 20); err != nil { // 25MB cap — raised for short video ad creatives
			writeJSONError(w, http.StatusBadRequest, "file too large or invalid form (25MB max)")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "no file provided (expected form field 'file')")
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to read upload")
			return
		}

		objectPath := buildObjectPath(header.Filename)
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		publicURL, err := uploadToSupabaseStorage(supabaseURL, serviceKey, bucket, objectPath, contentType, data)
		if err != nil {
			log.Printf("admin: media upload: %v", err)
			writeJSONError(w, http.StatusBadGateway, "upload to storage failed")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"url": publicURL})
	}
}

// buildObjectPath produces a collision-safe path: uploads/<unix-ms>-<8
// random hex chars>-<sanitized original filename>, so re-uploading a
// file with the same name never overwrites a live ad/logo mid-use.
func buildObjectPath(originalFilename string) string {
	ext := filepath.Ext(originalFilename)
	base := strings.TrimSuffix(filepath.Base(originalFilename), ext)
	base = unsafeFilenameChars.ReplaceAllString(base, "-")
	if base == "" {
		base = "file"
	}
	randSuffix := make([]byte, 4)
	_, _ = rand.Read(randSuffix)
	return fmt.Sprintf("uploads/%d-%s-%s%s", time.Now().UnixMilli(), hex.EncodeToString(randSuffix), base, ext)
}

func uploadToSupabaseStorage(supabaseURL, serviceKey, bucket, objectPath, contentType string, data []byte) (string, error) {
	uploadURL := fmt.Sprintf("%s/storage/v1/object/%s/%s", supabaseURL, bucket, objectPath)
	req, err := http.NewRequest(http.MethodPost, uploadURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+serviceKey)
	req.Header.Set("apikey", serviceKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "true")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("storage upload returned %d: %s", resp.StatusCode, string(body))
	}

	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", supabaseURL, bucket, objectPath), nil
}
