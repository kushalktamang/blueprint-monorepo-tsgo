package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/validation"
	"github.com/newrelic/go-agent/v3/integrations/nrpkgerrors"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// Handler provides base functionality for all handlers
type Handler struct {
	server *server.Server
}

// NewHandler creates a new base handler
func NewHandler(server *server.Server) Handler {
	return Handler{server: server}
}

// HandlerFunc represents a typed handler function that processes a request and returns a response
type HandlerFunc[Req validation.Validatable, Res any] func(ctx context.Context, req Req) (Res, error)

// HandlerFuncNoContent represents a typed handler function that processes a request without returning content
type HandlerFuncNoContent[Req validation.Validatable] func(ctx context.Context, req Req) error

// ResponseHandler defines the interface for handling different response types
type ResponseHandler interface {
	Handle(w http.ResponseWriter, result any) error
	GetOperation() string
	AddAttributes(txn *newrelic.Transaction, result interface{})
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// JSONResponseHandler handles JSON responses
type JSONResponseHandler struct {
	status int
}

func (h JSONResponseHandler) Handle(w http.ResponseWriter, result any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(h.status)
	return json.NewEncoder(w).Encode(result)
}

func (h JSONResponseHandler) GetOperation() string {
	return "handler"
}

func (h JSONResponseHandler) AddAttributes(txn *newrelic.Transaction, result any) {
	// http.status_code is already set by tracing middleware
}

// NoContentResponseHandler handles no-content responses
type NoContentResponseHandler struct {
	status int
}

func (h NoContentResponseHandler) Handle(w http.ResponseWriter, _ any) error {
	w.WriteHeader(h.status)
	return nil
}

func (h NoContentResponseHandler) GetOperation() string {
	return "handler_no_content"
}

func (h NoContentResponseHandler) AddAttributes(txn *newrelic.Transaction, result interface{}) {
	// http.status_code is already set by tracing middleware
}

// FileResponseHandler handles file responses
type FileResponseHandler struct {
	status      int
	filename    string
	contentType string
}

func (h FileResponseHandler) Handle(w http.ResponseWriter, result any) error {
	data, ok := result.([]byte)
	if !ok {
		return fmt.Errorf("file handler expected []byte, got %T", result)
	}
	// FormatMediaType quotes and escapes the filename (blocks header injection)
	w.Header().Set("Content-Disposition",
		mime.FormatMediaType("attachment", map[string]string{"filename": h.filename}))
	w.Header().Set("Content-Type", h.contentType)
	w.WriteHeader(h.status)
	_, err := w.Write(data)
	return err
}

func (h FileResponseHandler) AddAttributes(txn *newrelic.Transaction, result any) {
	if txn != nil {
		// http.status_code is already set by tracing middleware
		txn.AddAttribute("file.name", h.filename)
		txn.AddAttribute("file.content_type", h.contentType)
		if data, ok := result.([]byte); ok {
			txn.AddAttribute("file.size_bytes", len(data))
		}
	}
}

func (h FileResponseHandler) GetOperation() string {
	return "handler_file"
}

// handleRequest is the unified handler function that eliminates code duplication
func handleRequest[Req validation.Validatable](
	log *slog.Logger,
	w http.ResponseWriter,
	r *http.Request,
	handler func(ctx context.Context, req Req) (any, error),
	responseHandler ResponseHandler,
) {
	start := time.Now()
	ctx := r.Context()
	route := r.Pattern

	txn := newrelic.FromContext(ctx)
	if txn != nil {
		txn.AddAttribute("handler.name", route)
		responseHandler.AddAttributes(txn, nil)
	}

	// Get context-enhanced logger
	loggerBuilder := log.With(
		slog.String("operation", responseHandler.GetOperation()),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("route", route),
	)

	// Add file-specific fields to logger if it's a file handler
	if fileHandler, ok := responseHandler.(FileResponseHandler); ok {
		loggerBuilder = loggerBuilder.With(slog.String("filename", fileHandler.filename), slog.String("content_type", fileHandler.contentType))
	}

	loggerBuilder.InfoContext(ctx, "handling request")

	// Fresh request value per call, never shared between requests
	var req Req
	validationStart := time.Now()

	status := http.StatusUnprocessableEntity
	err := decodeBody(w, r, &req)
	if err != nil {
		status = http.StatusBadRequest
	} else {
		err = req.Validate()
	}
	validationDuration := time.Since(validationStart)

	if err != nil {
		loggerBuilder.ErrorContext(ctx, "request validation failed",
			slog.Any("error", err), slog.Duration("validation_duration", validationDuration))
		if txn != nil {
			txn.NoticeError(nrpkgerrors.Wrap(err))
			txn.AddAttribute("validation.status", "failed")
			txn.AddAttribute("validation.duration_ms", validationDuration.Milliseconds())
		}
		_ = writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	if txn != nil {
		txn.AddAttribute("validation.status", "success")
		txn.AddAttribute("validation.duration_ms", validationDuration.Milliseconds())
	}

	loggerBuilder.DebugContext(ctx, "request validation successful", slog.Duration("validation_duration", validationDuration))

	handlerStart := time.Now()
	result, err := handler(ctx, req)
	handlerDuration := time.Since(handlerStart)
	totalDuration := time.Since(start)

	if err != nil {
		loggerBuilder.ErrorContext(ctx, "handler execution failed",
			slog.Any("error", err),
			slog.Duration("handler_duration", handlerDuration),
			slog.Duration("total_duration", totalDuration))
		if txn != nil {
			txn.NoticeError(nrpkgerrors.Wrap(err))
			txn.AddAttribute("handler.status", "error")
			txn.AddAttribute("handler.duration_ms", handlerDuration.Milliseconds())
			txn.AddAttribute("total.duration_ms", totalDuration.Milliseconds())
		}
		_ = writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if txn != nil {
		txn.AddAttribute("handler.status", "success")
		txn.AddAttribute("handler.duration_ms", handlerDuration.Milliseconds())
		txn.AddAttribute("total.duration_ms", totalDuration.Milliseconds())
		responseHandler.AddAttributes(txn, result)
	}

	loggerBuilder.InfoContext(ctx, "request completed successfully",
		slog.Duration("handler_duration", handlerDuration),
		slog.Duration("validation_duration", validationDuration),
		slog.Duration("total_duration", totalDuration))

	if err := responseHandler.Handle(w, result); err != nil {
		// Headers may already be sent, so logging is all we can do
		loggerBuilder.ErrorContext(ctx, "failed to write response", slog.Any("error", err))
	}
}

// decodeBody skips empty bodies (GET/DELETE) and caps size at 1 MiB.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil || r.ContentLength == 0 {
		return nil
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request body must contain a single JSON value")
	}
	return nil
}

func Handle[Req validation.Validatable, Res any](
	h Handler,
	handler HandlerFunc[Req, Res],
	status int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleRequest(h.server.Logger, w, r, func(ctx context.Context, req Req) (any, error) {
			return handler(ctx, req)
		}, JSONResponseHandler{status: status})
	}
}

func HandleFile[Req validation.Validatable](
	h Handler,
	handler HandlerFunc[Req, []byte],
	status int,
	filename string,
	contentType string,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleRequest(h.server.Logger, w, r, func(ctx context.Context, req Req) (any, error) {
			return handler(ctx, req)
		}, FileResponseHandler{
			status:      status,
			filename:    filename,
			contentType: contentType,
		})
	}
}

// HandleNoContent wraps a handler with validation, error handling, logging, metrics, and tracing for endpoints that don't return content
func HandleNoContent[Req validation.Validatable](
	h Handler,
	handler HandlerFuncNoContent[Req],
	status int,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handleRequest(h.server.Logger, w, r, func(ctx context.Context, req Req) (any, error) {
			return nil, handler(ctx, req)
		}, NoContentResponseHandler{status: status})
	}
}
