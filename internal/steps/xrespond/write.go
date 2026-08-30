// Package xrespond serializes payloads and writes final HTTP headers and status codes.
package xrespond

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func sanitizeHeaders(s string) string {
	return strings.TrimSpace(strings.NewReplacer("\r", "", "\n", "").Replace(s))
}

func isJSONContentType(contentType string) bool {
	if contentType == "" {
		return true
	}
	lower := strings.ToLower(contentType)
	return strings.Contains(lower, "application/json") || strings.Contains(lower, "+json")
}

// Write sets sanitized headers, writes the status code, and serializes the response body.
func Write(w http.ResponseWriter, status int, headers map[string]string, body any) error {
	hasContentType := false

	for k, v := range headers {
		cleanKey := sanitizeHeaders(k)
		cleanVal := sanitizeHeaders(v)
		if cleanKey == "" {
			continue
		}
		if strings.EqualFold(cleanKey, "Content-Type") {
			hasContentType = true
		}
		w.Header().Set(cleanKey, cleanVal)
	}

	if !hasContentType && body != nil && status != http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(status)

	if body == nil || status == http.StatusNoContent {
		return nil
	}

	contentType := w.Header().Get("Content-Type")
	if isJSONContentType(contentType) {
		return json.NewEncoder(w).Encode(body)
	}

	// Write raw bytes/strings for non-JSON content types (e.g. text/plain, text/html, application/xml)
	switch b := body.(type) {
	case string:
		if _, err := w.Write([]byte(b)); err != nil {
			return fmt.Errorf("failed to write response string: %w", err)
		}
		return nil
	case []byte:
		if _, err := w.Write(b); err != nil {
			return fmt.Errorf("failed to write response bytes: %w", err)
		}
		return nil
	default:
		if _, err := fmt.Fprint(w, b); err != nil {
			return fmt.Errorf("failed to write response body: %w", err)
		}
		return nil
	}
}
