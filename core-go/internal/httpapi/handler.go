package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"roller_hoops/core-go/internal/db"
)

type Handler struct {
	log  zerolog.Logger
	pool *db.Pool
}

func NewHandler(log zerolog.Logger, pool *db.Pool) *Handler {
	return &Handler{log: log, pool: pool}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(h.accessLog)

	// Health
	r.Get("/healthz", h.handleHealthz)
	r.Get("/readyz", h.handleReadyZ)

	// API
	r.Route("/api", func(r chi.Router) {
		r.Route("/v1", func(r chi.Router) {
			r.Route("/devices", func(r chi.Router) {
				r.Get("/", h.handleListDevices)
				r.Post("/", h.handleCreateDevice)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", h.handleGetDevice)
					r.Put("/", h.handleUpdateDevice)
				})
			})

			r.Route("/discovery", func(r chi.Router) {
				r.Post("/run", h.handleDiscoveryRun)
				r.Get("/status", h.handleDiscoveryStatus)
			})
		})
	})

	return r
}

func (h *Handler) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		h.log.Info().
			Str("request_id", middleware.GetReqID(r.Context())).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", ww.Status()).
			Int("bytes", ww.BytesWritten()).
			Int64("duration_ms", time.Since(start).Milliseconds()).
			Msg("http_request")
	})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, code, msg string, details map[string]any) {
	resp := map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": msg,
		},
	}
	if details != nil {
		resp["error"].(map[string]any)["details"] = details
	}
	h.writeJSON(w, status, resp)
}

func decodeJSONStrict(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleReadyZ(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if h.pool != nil {
		if err := h.pool.Ping(ctx); err != nil {
			h.writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database not ready", map[string]any{"error": err.Error()})
			return
		}
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"ready": true})
}

type device struct {
	ID          string  `json:"id"`
	DisplayName *string `json:"display_name,omitempty"`
}

type deviceCreate struct {
	DisplayName *string `json:"display_name,omitempty"`
}

type deviceUpdate struct {
	DisplayName *string `json:"display_name,omitempty"`
}

func (h *Handler) handleListDevices(w http.ResponseWriter, r *http.Request) {
	// v1 stub: return empty list until sqlc queries are wired.
	h.writeJSON(w, http.StatusOK, []device{})
}

func (h *Handler) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var req deviceCreate
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}

	// v1 stub: no DB insert yet.
	resp := device{ID: "00000000-0000-0000-0000-000000000000", DisplayName: req.DisplayName}
	h.writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// v1 stub: always not found.
	h.writeError(w, http.StatusNotFound, "not_found", "device not found", map[string]any{"id": id})
}

func (h *Handler) handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req deviceUpdate
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}

	// v1 stub: echo payload.
	resp := device{ID: id, DisplayName: req.DisplayName}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleDiscoveryRun(w http.ResponseWriter, r *http.Request) {
	// v1 stub: accept request.
	h.writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted"})
}

func (h *Handler) handleDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	// v1 stub.
	h.writeJSON(w, http.StatusOK, map[string]any{"status": "idle"})
}
