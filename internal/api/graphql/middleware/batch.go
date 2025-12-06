package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"prometheus/pkg/logger"
)

// BatchMiddleware handles GraphQL batch requests (array of queries)
type BatchMiddleware struct {
	log *logger.Logger
}

// NewBatchMiddleware creates a new batch middleware
func NewBatchMiddleware(log *logger.Logger) *BatchMiddleware {
	return &BatchMiddleware{
		log: log.With("component", "graphql_batch"),
	}
}

// Handler wraps the GraphQL handler to support batch requests
// GraphQL spec allows sending multiple operations in one HTTP request as JSON array
func (m *BatchMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.log.Infow("Batch middleware invoked",
			"method", r.Method,
			"path", r.URL.Path,
			"content_type", r.Header.Get("Content-Type"),
			"cookie_header", r.Header.Get("Cookie"),
			"cookies_count", len(r.Cookies()),
		)

		// Only handle POST requests with JSON body
		if r.Method != http.MethodPost {
			m.log.Debug("Not POST request, passing through")
			next.ServeHTTP(w, r)
			return
		}

		// Read body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			m.log.Errorw("Failed to read request body", "error", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		m.log.Debugw("Read request body", "length", len(body), "preview", string(body[:min(len(body), 100)]))

		// Try to detect if body is array or single object
		trimmedBody := bytes.TrimSpace(body)
		if len(trimmedBody) == 0 {
			http.Error(w, "Empty request body", http.StatusBadRequest)
			return
		}

		// Check if body starts with '[' - it's a batch request
		if trimmedBody[0] != '[' {
			// Single request - pass through to gqlgen handler
			m.log.Debug("Single request detected, passing through")
			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
			return
		}

		// Batch request - handle multiple queries
		m.log.Infow("🔵 Batch request detected", "body_length", len(body))

		// Parse batch request
		var batchRequests []json.RawMessage
		if err := json.Unmarshal(body, &batchRequests); err != nil {
			m.log.Errorw("Failed to parse batch request", "error", err)
			http.Error(w, "Invalid batch request format", http.StatusBadRequest)
			return
		}

		if len(batchRequests) == 0 {
			http.Error(w, "Empty batch request", http.StatusBadRequest)
			return
		}

		m.log.Infow("Processing batch request", "count", len(batchRequests))

		// Process each request individually
		batchResponses := make([]json.RawMessage, len(batchRequests))
		for i, reqBody := range batchRequests {
			// Create a new request for each operation
			req := r.Clone(r.Context())
			req.Body = io.NopCloser(bytes.NewReader(reqBody))

			// Create response recorder to capture handler output
			recorder := httptest.NewRecorder()

			// Execute handler for this request
			next.ServeHTTP(recorder, req)

			// Capture response body
			respBody := recorder.Body.Bytes()
			batchResponses[i] = json.RawMessage(respBody)

			m.log.Debugw("Batch item processed",
				"index", i,
				"status", recorder.Code,
				"body_length", len(respBody),
			)
		}

		// Return batch response as JSON array
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(batchResponses); err != nil {
			m.log.Errorw("Failed to encode batch response", "error", err)
			return
		}

		m.log.Infow("Batch request completed", "count", len(batchResponses))
	})
}
