package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"

	"roller_hoops/core-go/internal/db"
	"roller_hoops/core-go/internal/sqlcgen"
)

type Handler struct {
	log       zerolog.Logger
	pool      *db.Pool
	devices   deviceQueries
	discovery discoveryQueries
}

type deviceQueries interface {
	ListDevices(ctx context.Context) ([]sqlcgen.Device, error)
	GetDevice(ctx context.Context, id string) (sqlcgen.Device, error)
	CreateDevice(ctx context.Context, displayName *string) (sqlcgen.Device, error)
	UpdateDevice(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error)
	UpsertDeviceMetadata(ctx context.Context, arg sqlcgen.UpsertDeviceMetadataParams) (sqlcgen.DeviceMetadata, error)
}

type discoveryQueries interface {
	InsertDiscoveryRun(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	UpdateDiscoveryRun(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	GetLatestDiscoveryRun(ctx context.Context) (sqlcgen.DiscoveryRun, error)
	GetDiscoveryRun(ctx context.Context, id string) (sqlcgen.DiscoveryRun, error)
	InsertDiscoveryRunLog(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error
}

func NewHandler(log zerolog.Logger, pool *db.Pool) *Handler {
	var dq deviceQueries
	var drq discoveryQueries
	if pool != nil {
		q := pool.Queries()
		dq = q
		drq = q
	}
	return &Handler{log: log, pool: pool, devices: dq, discovery: drq}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(h.ensureResponseRequestID)
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

func (h *Handler) ensureResponseRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rid := middleware.GetReqID(r.Context()); rid != "" {
			// Echo request id so clients (and upstream proxies) can correlate logs.
			w.Header().Set("X-Request-ID", rid)
		}
		next.ServeHTTP(w, r)
	})
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
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("unexpected extra data after JSON body")
		}
		return err
	}
	return nil
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleReadyZ(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if h.pool == nil {
		h.writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database not configured", nil)
		return
	}

	if err := h.pool.Ping(ctx); err != nil {
		h.writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database not ready", map[string]any{"error": err.Error()})
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"ready": true})
}

type device struct {
	ID          string          `json:"id"`
	DisplayName *string         `json:"display_name,omitempty"`
	Metadata    *deviceMetadata `json:"metadata,omitempty"`
}

type deviceMetadata struct {
	Owner    *string `json:"owner,omitempty"`
	Location *string `json:"location,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

type deviceMetadataBody struct {
	Owner    *string `json:"owner,omitempty"`
	Location *string `json:"location,omitempty"`
	Notes    *string `json:"notes,omitempty"`
}

type deviceCreate struct {
	DisplayName *string             `json:"display_name,omitempty"`
	Metadata    *deviceMetadataBody `json:"metadata,omitempty"`
}

type deviceUpdate struct {
	DisplayName *string             `json:"display_name,omitempty"`
	Metadata    *deviceMetadataBody `json:"metadata,omitempty"`
}

func (h *Handler) ensureDeviceQueries(w http.ResponseWriter) bool {
	if h.devices == nil {
		h.writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database not configured", nil)
		return false
	}
	return true
}

func toDevice(d sqlcgen.Device) device {
	var meta *deviceMetadata
	if d.Owner != nil || d.Location != nil || d.Notes != nil {
		meta = &deviceMetadata{
			Owner:    d.Owner,
			Location: d.Location,
			Notes:    d.Notes,
		}
	}

	return device{
		ID:          d.ID,
		DisplayName: d.DisplayName,
		Metadata:    meta,
	}
}

func normalizeMetadataBody(body *deviceMetadataBody) *deviceMetadataBody {
	if body == nil {
		return nil
	}

	trim := func(v *string) *string {
		if v == nil {
			return nil
		}
		s := strings.TrimSpace(*v)
		if s == "" {
			return nil
		}
		return &s
	}

	normalized := &deviceMetadataBody{
		Owner:    trim(body.Owner),
		Location: trim(body.Location),
		Notes:    trim(body.Notes),
	}

	if normalized.Owner == nil && normalized.Location == nil && normalized.Notes == nil {
		return nil
	}

	return normalized
}

type discoveryRun struct {
	ID          string         `json:"id"`
	Status      string         `json:"status"`
	Scope       *string        `json:"scope,omitempty"`
	Stats       map[string]any `json:"stats,omitempty"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	LastError   *string        `json:"last_error,omitempty"`
}

type discoveryStatus struct {
	Status    string        `json:"status"`
	LatestRun *discoveryRun `json:"latest_run,omitempty"`
}

type discoveryRunRequest struct {
	Scope *string `json:"scope,omitempty"`
}

func (h *Handler) ensureDiscoveryQueries(w http.ResponseWriter) bool {
	if h.discovery == nil {
		h.writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database not configured", nil)
		return false
	}
	return true
}

func toDiscoveryRun(dr sqlcgen.DiscoveryRun) discoveryRun {
	return discoveryRun{
		ID:          dr.ID,
		Status:      dr.Status,
		Scope:       dr.Scope,
		Stats:       dr.Stats,
		StartedAt:   dr.StartedAt,
		CompletedAt: dr.CompletedAt,
		LastError:   dr.LastError,
	}
}

func normalizeScope(scope *string) *string {
	if scope == nil {
		return nil
	}
	s := strings.TrimSpace(*scope)
	if s == "" {
		return nil
	}
	return &s
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "22P02"
	}
	return false
}

func (h *Handler) handleListDevices(w http.ResponseWriter, r *http.Request) {
	if !h.ensureDeviceQueries(w) {
		return
	}

	rows, err := h.devices.ListDevices(r.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("list devices failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list devices", nil)
		return
	}

	resp := make([]device, 0, len(rows))
	for _, d := range rows {
		resp = append(resp, toDevice(d))
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleCreateDevice(w http.ResponseWriter, r *http.Request) {
	var req deviceCreate
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}

	req.Metadata = normalizeMetadataBody(req.Metadata)

	if !h.ensureDeviceQueries(w) {
		return
	}

	ctx := r.Context()
	row, err := h.devices.CreateDevice(ctx, req.DisplayName)
	if err != nil {
		h.log.Error().Err(err).Msg("create device failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to create device", nil)
		return
	}

	if req.Metadata != nil {
		meta, err := h.devices.UpsertDeviceMetadata(ctx, sqlcgen.UpsertDeviceMetadataParams{
			DeviceID: row.ID,
			Owner:    req.Metadata.Owner,
			Location: req.Metadata.Location,
			Notes:    req.Metadata.Notes,
		})
		if err != nil {
			h.log.Error().Err(err).Str("device_id", row.ID).Msg("failed to upsert metadata")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to create device metadata", nil)
			return
		}

		row.Owner = meta.Owner
		row.Location = meta.Location
		row.Notes = meta.Notes
	}

	h.writeJSON(w, http.StatusCreated, toDevice(row))
}

func (h *Handler) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.ensureDeviceQueries(w) {
		return
	}

	row, err := h.devices.GetDevice(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			h.writeError(w, http.StatusNotFound, "not_found", "device not found", map[string]any{"id": id})
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("get device failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to fetch device", nil)
		}
		return
	}

	h.writeJSON(w, http.StatusOK, toDevice(row))
}

func (h *Handler) handleUpdateDevice(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req deviceUpdate
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}

	req.Metadata = normalizeMetadataBody(req.Metadata)

	if !h.ensureDeviceQueries(w) {
		return
	}

	ctx := r.Context()
	row, err := h.devices.UpdateDevice(ctx, sqlcgen.UpdateDeviceParams{
		ID:          id,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			h.writeError(w, http.StatusNotFound, "not_found", "device not found", map[string]any{"id": id})
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("update device failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to update device", nil)
		}
		return
	}

	if req.Metadata != nil {
		meta, err := h.devices.UpsertDeviceMetadata(ctx, sqlcgen.UpsertDeviceMetadataParams{
			DeviceID: id,
			Owner:    req.Metadata.Owner,
			Location: req.Metadata.Location,
			Notes:    req.Metadata.Notes,
		})
		if err != nil {
			h.log.Error().Err(err).Str("id", id).Msg("update device metadata failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to update device metadata", nil)
			return
		}

		row.Owner = meta.Owner
		row.Location = meta.Location
		row.Notes = meta.Notes
	}

	h.writeJSON(w, http.StatusOK, toDevice(row))
}

func (h *Handler) handleDiscoveryRun(w http.ResponseWriter, r *http.Request) {
	if !h.ensureDiscoveryQueries(w) {
		return
	}

	var req discoveryRunRequest
	if r.ContentLength > 0 {
		if err := decodeJSONStrict(r, &req); err != nil {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
			return
		}
	}
	req.Scope = normalizeScope(req.Scope)

	run, err := h.discovery.InsertDiscoveryRun(r.Context(), sqlcgen.InsertDiscoveryRunParams{
		Status: "queued",
		Scope:  req.Scope,
		Stats:  map[string]any{"stage": "queued"},
	})
	if err != nil {
		h.log.Error().Err(err).Msg("failed to create discovery run")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to start discovery", nil)
		return
	}

	h.writeJSON(w, http.StatusAccepted, toDiscoveryRun(run))
}

func (h *Handler) handleDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	if !h.ensureDiscoveryQueries(w) {
		return
	}

	run, err := h.discovery.GetLatestDiscoveryRun(r.Context())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.writeJSON(w, http.StatusOK, discoveryStatus{Status: "idle"})
			return
		}

		h.log.Error().Err(err).Msg("failed to fetch discovery status")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to load discovery status", nil)
		return
	}

	latest := toDiscoveryRun(run)
	h.writeJSON(w, http.StatusOK, discoveryStatus{
		Status:    run.Status,
		LatestRun: &latest,
	})
}
