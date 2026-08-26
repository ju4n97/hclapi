package xrespond

import (
	"encoding/json"
	"net/http"

	"github.com/ju4n97/hclapi/internal/parser"
)

// Write writes the HTTP status code, headers and payload to the ResponseWriter.
func Write(w http.ResponseWriter, cfg *parser.RespondStepConfig, lastResult any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(cfg.Status)

	// Explicit static body defined in HCL
	if cfg.Body != nil {
		_, err := w.Write([]byte(*cfg.Body))
		return err
	}

	// Output of the immediately preceding step
	if lastResult != nil {
		return json.NewEncoder(w).Encode(lastResult)
	}

	return nil
}
