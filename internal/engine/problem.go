package engine

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/ju4n97/hclapi/internal/config"
	"github.com/ju4n97/hclapi/internal/problem"
)

// ErrorResponder centralizes RFC 9457 error formatting and response serialization.
type ErrorResponder struct {
	problemConfig config.Problem
	handler       problem.Handler
	logger        *slog.Logger
}

// NewErrorResponder initializes an error responder with the given problem configuration.
func NewErrorResponder(p config.Problem, h problem.Handler, logger *slog.Logger) *ErrorResponder {
	if h == nil {
		h = problem.DefaultHandler
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &ErrorResponder{
		problemConfig: p,
		handler:       h,
		logger:        logger,
	}
}

// Respond formats any error into an RFC 9457 Problem Details payload and sends it to the client.
func (r *ErrorResponder) Respond(w http.ResponseWriter, req *http.Request, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		r.logger.WarnContext(req.Context(), "request payload too large", "error", maxBytesErr, "path", req.URL.Path)
		r.handler(w, req, problem.Problem{
			Type:     r.problemConfig.FormatType("payload-too-large"),
			Title:    "Request Entity Too Large",
			Status:   http.StatusRequestEntityTooLarge,
			Detail:   "request body exceeded maximum allowed size",
			Instance: req.URL.Path,
		})
		return
	}

	var p problem.Problem
	if errors.As(err, &p) {
		if p.Status == 0 {
			p.Status = http.StatusInternalServerError
		}
		if p.Title == "" {
			p.Title = http.StatusText(p.Status)
			if p.Title == "" {
				p.Title = "Error"
			}
		}
		if p.Type == "" {
			p.Type = r.problemConfig.FormatType(problem.Slugify(p.Title))
		}
		if p.Instance == "" {
			p.Instance = req.URL.Path
		}

		if p.Status >= 500 {
			r.logger.ErrorContext(req.Context(), "step execution failed", "error", p, "path", req.URL.Path)
		} else {
			r.logger.WarnContext(req.Context(), "step rejected request", "status", p.Status, "title", p.Title, "path", req.URL.Path)
		}

		r.handler(w, req, p)
		return
	}

	r.logger.ErrorContext(req.Context(), "pipeline execution failed", "error", err, "path", req.URL.Path)
	r.handler(w, req, problem.Problem{
		Type:     r.problemConfig.FormatType("pipeline-execution-failed"),
		Title:    "Pipeline Execution Error",
		Status:   http.StatusInternalServerError,
		Detail:   err.Error(),
		Instance: req.URL.Path,
	})
}
