package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strconv"
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
	inventory inventoryQueries
}

type deviceQueries interface {
	ListDevices(ctx context.Context) ([]sqlcgen.Device, error)
	GetDevice(ctx context.Context, id string) (sqlcgen.Device, error)
	CreateDevice(ctx context.Context, displayName *string) (sqlcgen.Device, error)
	UpdateDevice(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error)
	UpsertDeviceMetadata(ctx context.Context, arg sqlcgen.UpsertDeviceMetadataParams) (sqlcgen.DeviceMetadata, error)
	ListDeviceNameCandidates(ctx context.Context, deviceID string) ([]sqlcgen.DeviceNameCandidate, error)
	ListDeviceChangeEvents(ctx context.Context, arg sqlcgen.ListDeviceChangeEventsParams) ([]sqlcgen.DeviceChangeEvent, error)
	ListDeviceChangeEventsForDevice(ctx context.Context, arg sqlcgen.ListDeviceChangeEventsForDeviceParams) ([]sqlcgen.DeviceChangeEvent, error)
}

type discoveryQueries interface {
	InsertDiscoveryRun(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	UpdateDiscoveryRun(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	GetLatestDiscoveryRun(ctx context.Context) (sqlcgen.DiscoveryRun, error)
	GetDiscoveryRun(ctx context.Context, id string) (sqlcgen.DiscoveryRun, error)
	InsertDiscoveryRunLog(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error
	ListDiscoveryRuns(ctx context.Context, arg sqlcgen.ListDiscoveryRunsParams) ([]sqlcgen.DiscoveryRun, error)
	ListDiscoveryRunLogs(ctx context.Context, arg sqlcgen.ListDiscoveryRunLogsParams) ([]sqlcgen.DiscoveryRunLog, error)
}

type inventoryQueries interface {
	FindDeviceIDByIP(ctx context.Context, ip string) (string, error)
	CreateDevice(ctx context.Context, displayName *string) (sqlcgen.Device, error)
	SetDeviceDisplayNameIfUnset(ctx context.Context, arg sqlcgen.SetDeviceDisplayNameIfUnsetParams) (int64, error)
	UpsertDeviceIP(ctx context.Context, arg sqlcgen.UpsertDeviceIPParams) error
	UpsertDeviceMetadataFillBlank(ctx context.Context, arg sqlcgen.UpsertDeviceMetadataParams) (sqlcgen.DeviceMetadata, error)
}

func NewHandler(log zerolog.Logger, pool *db.Pool) *Handler {
	var dq deviceQueries
	var drq discoveryQueries
	var iq inventoryQueries
	if pool != nil {
		q := pool.Queries()
		dq = q
		drq = q
		iq = q
	}
	return &Handler{log: log, pool: pool, devices: dq, discovery: drq, inventory: iq}
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
				r.Get("/changes", h.handleListDeviceChangeEvents)
				r.Get("/export", h.handleExportDevices)
				r.Post("/", h.handleCreateDevice)
				r.Post("/import", h.handleImportDevices)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", h.handleGetDevice)
					r.Get("/name-candidates", h.handleListDeviceNameCandidates)
					r.Get("/history", h.handleDeviceHistory)
					r.Put("/", h.handleUpdateDevice)
				})
			})

			r.Route("/discovery", func(r chi.Router) {
				r.Post("/run", h.handleDiscoveryRun)
				r.Get("/status", h.handleDiscoveryStatus)
				r.Route("/runs", func(r chi.Router) {
					r.Get("/", h.handleListDiscoveryRuns)
					r.Route("/{id}", func(r chi.Router) {
						r.Get("/", h.handleGetDiscoveryRun)
						r.Get("/logs", h.handleListDiscoveryRunLogs)
					})
				})
			})

			r.Route("/inventory", func(r chi.Router) {
				r.Post("/netbox/import", h.handleImportNetBox)
				r.Post("/nautobot/import", h.handleImportNautobot)
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

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
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

type deviceNameCandidate struct {
	Name       string  `json:"name"`
	Source     string  `json:"source"`
	Address    *string `json:"address,omitempty"`
	ObservedAt string  `json:"observed_at"`
}

type deviceChangeEventResponse struct {
	EventID  string         `json:"event_id"`
	DeviceID string         `json:"device_id"`
	EventAt  time.Time      `json:"event_at"`
	Kind     string         `json:"kind"`
	Summary  string         `json:"summary"`
	Details  map[string]any `json:"details,omitempty"`
}

type deviceChangeEventsResponse struct {
	Events []deviceChangeEventResponse `json:"events"`
	Cursor *string                     `json:"cursor,omitempty"`
}

type discoveryRunPage struct {
	Runs   []discoveryRun `json:"runs"`
	Cursor *string        `json:"cursor,omitempty"`
}

type discoveryRunLogEntry struct {
	ID        int64     `json:"id"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type discoveryRunLogPage struct {
	Logs   []discoveryRunLogEntry `json:"logs"`
	Cursor *string                `json:"cursor,omitempty"`
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

type importDevice struct {
	ID          *string             `json:"id,omitempty"`
	DisplayName *string             `json:"display_name,omitempty"`
	Metadata    *deviceMetadataBody `json:"metadata,omitempty"`
}

type importDevicesRequest struct {
	Devices []importDevice `json:"devices"`
}

type importDevicesResult struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
}

type inventoryImportRequest struct {
	Payload json.RawMessage `json:"payload"`
}

type inventoryImportResult struct {
	Created         int `json:"created"`
	MatchedExisting int `json:"matched_existing"`
	IPWritten       int `json:"ip_written"`
	MetadataWritten int `json:"metadata_written"`
	Skipped         int `json:"skipped"`
}

func (h *Handler) ensureDeviceQueries(w http.ResponseWriter) bool {
	if h.devices == nil {
		h.writeError(w, http.StatusServiceUnavailable, "db_unavailable", "database not configured", nil)
		return false
	}
	return true
}

func (h *Handler) ensureInventoryQueries(w http.ResponseWriter) bool {
	if h.inventory == nil {
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

func normalizeID(id *string) *string {
	if id == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*id)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (h *Handler) listDevicesPayload(ctx context.Context) ([]device, error) {
	rows, err := h.devices.ListDevices(ctx)
	if err != nil {
		return nil, err
	}
	resp := make([]device, 0, len(rows))
	for _, d := range rows {
		resp = append(resp, toDevice(d))
	}
	return resp, nil
}

func (h *Handler) persistImportMetadata(ctx context.Context, deviceID string, meta *deviceMetadataBody) error {
	if meta == nil {
		return nil
	}
	_, err := h.devices.UpsertDeviceMetadata(ctx, sqlcgen.UpsertDeviceMetadataParams{
		DeviceID: deviceID,
		Owner:    meta.Owner,
		Location: meta.Location,
		Notes:    meta.Notes,
	})
	if err != nil {
		h.log.Error().Err(err).Str("device_id", deviceID).Msg("failed to import metadata")
	}
	return err
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

func validateAndCanonicalizeScope(scope *string) (*string, error) {
	if scope == nil {
		return nil, nil
	}

	s := strings.TrimSpace(*scope)
	if s == "" {
		return nil, nil
	}

	if p, err := netip.ParsePrefix(s); err == nil {
		c := p.String()
		return &c, nil
	}
	if a, err := netip.ParseAddr(s); err == nil {
		c := a.String()
		return &c, nil
	}

	return nil, errors.New("scope must be a CIDR prefix (e.g. 10.0.0.0/24) or a single IP (e.g. 10.0.0.5)")
}

func isInvalidUUID(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "22P02"
	}
	return false
}

func encodeCursor(ts time.Time, id string) string {
	payload := fmt.Sprintf("%s|%s", ts.UTC().Format(time.RFC3339Nano), id)
	return base64.RawURLEncoding.EncodeToString([]byte(payload))
}

func decodeCursor(value string) (time.Time, string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor: %w", err)
	}
	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", errors.New("invalid cursor")
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor timestamp: %w", err)
	}
	return ts, parts[1], nil
}

func parseLimitParam(value string, defaultVal, max int) (int, error) {
	if value == "" {
		return defaultVal, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid limit")
	}
	if limit <= 0 {
		return 0, fmt.Errorf("limit must be positive")
	}
	if limit > max {
		limit = max
	}
	return limit, nil
}

func parseSinceParam(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, fmt.Errorf("invalid since timestamp: %w", err)
	}
	return &ts, nil
}

func toDeviceChangeEventResponse(event sqlcgen.DeviceChangeEvent) deviceChangeEventResponse {
	return deviceChangeEventResponse{
		EventID:  event.EventID,
		DeviceID: event.DeviceID,
		EventAt:  event.EventAt,
		Kind:     event.Kind,
		Summary:  event.Summary,
		Details:  event.Details,
	}
}

func toDiscoveryRunLogEntry(log sqlcgen.DiscoveryRunLog) discoveryRunLogEntry {
	return discoveryRunLogEntry{
		ID:        log.ID,
		Level:     log.Level,
		Message:   log.Message,
		CreatedAt: log.CreatedAt,
	}
}

func (h *Handler) handleListDevices(w http.ResponseWriter, r *http.Request) {
	if !h.ensureDeviceQueries(w) {
		return
	}

	resp, err := h.listDevicesPayload(r.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("list devices failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list devices", nil)
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleExportDevices(w http.ResponseWriter, r *http.Request) {
	if !h.ensureDeviceQueries(w) {
		return
	}

	resp, err := h.listDevicesPayload(r.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("export devices failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to export devices", nil)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\"roller_hoops_devices.json\"")
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

func (h *Handler) handleImportDevices(w http.ResponseWriter, r *http.Request) {
	if !h.ensureDeviceQueries(w) {
		return
	}

	var req importDevicesRequest
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}

	if len(req.Devices) == 0 {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "no devices provided", nil)
		return
	}

	ctx := r.Context()
	result := importDevicesResult{}

	for _, entry := range req.Devices {
		meta := normalizeMetadataBody(entry.Metadata)
		if id := normalizeID(entry.ID); id != nil {
			row, err := h.devices.UpdateDevice(ctx, sqlcgen.UpdateDeviceParams{
				ID:          *id,
				DisplayName: entry.DisplayName,
			})
			if err != nil {
				switch {
				case errors.Is(err, pgx.ErrNoRows):
					h.writeError(w, http.StatusNotFound, "not_found", "device not found", map[string]any{"id": *id})
				case isInvalidUUID(err):
					h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": *id})
				default:
					h.log.Error().Err(err).Str("id", *id).Msg("update device failed during import")
					h.writeError(w, http.StatusInternalServerError, "db_error", "failed to update device", nil)
				}
				return
			}
			if err := h.persistImportMetadata(ctx, row.ID, meta); err != nil {
				h.writeError(w, http.StatusInternalServerError, "db_error", "failed to import device metadata", nil)
				return
			}
			result.Updated++
			continue
		}

		row, err := h.devices.CreateDevice(ctx, entry.DisplayName)
		if err != nil {
			h.log.Error().Err(err).Msg("create device failed during import")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to create device", nil)
			return
		}
		if err := h.persistImportMetadata(ctx, row.ID, meta); err != nil {
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to import device metadata", nil)
			return
		}
		result.Created++
	}

	h.writeJSON(w, http.StatusOK, result)
}

type netboxNameRef struct {
	Name string `json:"name"`
}

type netboxIPRef struct {
	Address string `json:"address"`
}

type netboxDevice struct {
	Name       string         `json:"name"`
	Comments   *string        `json:"comments,omitempty"`
	Site       *netboxNameRef `json:"site,omitempty"`
	Tenant     *netboxNameRef `json:"tenant,omitempty"`
	PrimaryIP  *netboxIPRef   `json:"primary_ip,omitempty"`
	PrimaryIP4 *netboxIPRef   `json:"primary_ip4,omitempty"`
	PrimaryIP6 *netboxIPRef   `json:"primary_ip6,omitempty"`
}

type netboxDeviceEnvelope struct {
	Results []netboxDevice `json:"results,omitempty"`
	Devices []netboxDevice `json:"devices,omitempty"`
}

func (h *Handler) handleImportNetBox(w http.ResponseWriter, r *http.Request) {
	h.handleImportInventoryPayload(w, r, "netbox")
}

func (h *Handler) handleImportNautobot(w http.ResponseWriter, r *http.Request) {
	h.handleImportInventoryPayload(w, r, "nautobot")
}

func (h *Handler) handleImportInventoryPayload(w http.ResponseWriter, r *http.Request, source string) {
	if !h.ensureInventoryQueries(w) {
		return
	}

	var req inventoryImportRequest
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}
	if len(req.Payload) == 0 {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "missing payload", nil)
		return
	}

	var env netboxDeviceEnvelope
	if err := json.Unmarshal(req.Payload, &env); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid payload json", map[string]any{"error": err.Error()})
		return
	}
	devs := env.Results
	if len(devs) == 0 {
		devs = env.Devices
	}
	if len(devs) == 0 {
		var direct []netboxDevice
		if err := json.Unmarshal(req.Payload, &direct); err == nil {
			devs = direct
		}
	}
	if len(devs) == 0 {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "payload contained no devices", map[string]any{"source": source})
		return
	}

	ctx := r.Context()
	res := inventoryImportResult{}

	for _, d := range devs {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			res.Skipped++
			continue
		}

		ip := firstNonEmptyNetboxIP(d.PrimaryIP4, d.PrimaryIP, d.PrimaryIP6)
		deviceID := ""
		if ip != nil {
			found, err := h.inventory.FindDeviceIDByIP(ctx, *ip)
			if err == nil && found != "" {
				deviceID = found
				res.MatchedExisting++
			}
		}

		if deviceID == "" {
			display := name
			row, err := h.inventory.CreateDevice(ctx, &display)
			if err != nil {
				h.log.Error().Err(err).Msg("inventory import create device failed")
				h.writeError(w, http.StatusInternalServerError, "db_error", "failed to import inventory devices", nil)
				return
			}
			deviceID = row.ID
			res.Created++
		} else {
			_, _ = h.inventory.SetDeviceDisplayNameIfUnset(ctx, sqlcgen.SetDeviceDisplayNameIfUnsetParams{
				ID:          deviceID,
				DisplayName: name,
			})
		}

		if deviceID == "" {
			res.Skipped++
			continue
		}

		if ip != nil {
			if err := h.inventory.UpsertDeviceIP(ctx, sqlcgen.UpsertDeviceIPParams{
				DeviceID: deviceID,
				IP:       *ip,
			}); err == nil {
				res.IPWritten++
			}
		}

		owner := normalizeStringPtr(fromNameRef(d.Tenant))
		location := normalizeStringPtr(fromNameRef(d.Site))
		notes := normalizeStringPtr(d.Comments)

		if owner != nil || location != nil || notes != nil {
			if _, err := h.inventory.UpsertDeviceMetadataFillBlank(ctx, sqlcgen.UpsertDeviceMetadataParams{
				DeviceID: deviceID,
				Owner:    owner,
				Location: location,
				Notes:    notes,
			}); err == nil {
				res.MetadataWritten++
			}
		}
	}

	h.writeJSON(w, http.StatusOK, res)
}

func fromNameRef(r *netboxNameRef) *string {
	if r == nil {
		return nil
	}
	s := strings.TrimSpace(r.Name)
	if s == "" {
		return nil
	}
	return &s
}

func normalizeStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func firstNonEmptyNetboxIP(refs ...*netboxIPRef) *string {
	for _, r := range refs {
		if r == nil {
			continue
		}
		addr := strings.TrimSpace(r.Address)
		if addr == "" {
			continue
		}
		if ip := stripPrefixLen(addr); ip != "" {
			return &ip
		}
	}
	return nil
}

func stripPrefixLen(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if i := strings.IndexByte(addr, '/'); i > 0 {
		addr = addr[:i]
	}
	if _, err := netip.ParseAddr(addr); err != nil {
		return ""
	}
	return addr
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

func (h *Handler) handleListDeviceNameCandidates(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.ensureDeviceQueries(w) {
		return
	}

	rows, err := h.devices.ListDeviceNameCandidates(r.Context(), id)
	if err != nil {
		switch {
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("list device name candidates failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list device name candidates", nil)
		}
		return
	}

	out := make([]deviceNameCandidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, deviceNameCandidate{
			Name:       row.Name,
			Source:     row.Source,
			Address:    row.Address,
			ObservedAt: row.ObservedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *Handler) handleListDeviceChangeEvents(w http.ResponseWriter, r *http.Request) {
	if !h.ensureDeviceQueries(w) {
		return
	}
	limitParam := r.URL.Query().Get("limit")
	limit, err := parseLimitParam(limitParam, 50, 100)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid limit", map[string]any{"error": err.Error()})
		return
	}
	var since *time.Time
	if sinceParam := r.URL.Query().Get("since"); sinceParam != "" {
		since, err = parseSinceParam(sinceParam)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid since timestamp", map[string]any{"error": err.Error()})
			return
		}
	}
	var beforeEventAt *time.Time
	var beforeEventID *string
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		ts, id, err := decodeCursor(cursor)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid cursor", map[string]any{"error": err.Error()})
			return
		}
		beforeEventAt = &ts
		beforeEventID = &id
		since = nil
	}
	rows, err := h.devices.ListDeviceChangeEvents(r.Context(), sqlcgen.ListDeviceChangeEventsParams{
		BeforeEventAt: beforeEventAt,
		BeforeEventID: beforeEventID,
		SinceEventAt:  since,
		Limit:         int32(limit + 1),
	})
	if err != nil {
		h.log.Error().Err(err).Msg("list device change events failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list device change events", nil)
		return
	}
	cursorOut := (*string)(nil)
	if len(rows) > limit {
		last := rows[limit-1]
		next := encodeCursor(last.EventAt, last.EventID)
		cursorOut = &next
		rows = rows[:limit]
	}
	resp := make([]deviceChangeEventResponse, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, toDeviceChangeEventResponse(row))
	}
	h.writeJSON(w, http.StatusOK, deviceChangeEventsResponse{Events: resp, Cursor: cursorOut})
}

func (h *Handler) handleDeviceHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.ensureDeviceQueries(w) {
		return
	}
	if _, err := h.devices.GetDevice(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			h.writeError(w, http.StatusNotFound, "not_found", "device not found", map[string]any{"id": id})
			return
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
			return
		default:
			h.log.Error().Err(err).Str("id", id).Msg("fetch device before history failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to fetch device history", nil)
			return
		}
	}
	limitParam := r.URL.Query().Get("limit")
	limit, err := parseLimitParam(limitParam, 50, 100)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid limit", map[string]any{"error": err.Error()})
		return
	}
	var beforeEventAt *time.Time
	var beforeEventID *string
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		ts, evID, err := decodeCursor(cursor)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid cursor", map[string]any{"error": err.Error()})
			return
		}
		beforeEventAt = &ts
		beforeEventID = &evID
	}
	rows, err := h.devices.ListDeviceChangeEventsForDevice(r.Context(), sqlcgen.ListDeviceChangeEventsForDeviceParams{
		DeviceID:      id,
		BeforeEventAt: beforeEventAt,
		BeforeEventID: beforeEventID,
		Limit:         int32(limit + 1),
	})
	if err != nil {
		h.log.Error().Err(err).Str("device_id", id).Msg("list device history failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list device history", nil)
		return
	}
	cursorOut := (*string)(nil)
	if len(rows) > limit {
		last := rows[limit-1]
		next := encodeCursor(last.EventAt, last.EventID)
		cursorOut = &next
		rows = rows[:limit]
	}
	resp := make([]deviceChangeEventResponse, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, toDeviceChangeEventResponse(row))
	}
	h.writeJSON(w, http.StatusOK, deviceChangeEventsResponse{Events: resp, Cursor: cursorOut})
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
	var err error
	req.Scope, err = validateAndCanonicalizeScope(req.Scope)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid discovery scope", map[string]any{"error": err.Error()})
		return
	}

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

func (h *Handler) handleListDiscoveryRuns(w http.ResponseWriter, r *http.Request) {
	if !h.ensureDiscoveryQueries(w) {
		return
	}
	limit, err := parseLimitParam(r.URL.Query().Get("limit"), 20, 200)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid limit", map[string]any{"error": err.Error()})
		return
	}
	var beforeStartedAt *time.Time
	var beforeID *string
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		ts, id, err := decodeCursor(cursor)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid cursor", map[string]any{"error": err.Error()})
			return
		}
		beforeStartedAt = &ts
		beforeID = &id
	}
	rows, err := h.discovery.ListDiscoveryRuns(r.Context(), sqlcgen.ListDiscoveryRunsParams{
		BeforeStartedAt: beforeStartedAt,
		BeforeID:        beforeID,
		Limit:           int32(limit + 1),
	})
	if err != nil {
		h.log.Error().Err(err).Msg("list discovery runs failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list discovery runs", nil)
		return
	}
	cursorOut := (*string)(nil)
	if len(rows) > limit {
		last := rows[limit-1]
		next := encodeCursor(last.StartedAt, last.ID)
		cursorOut = &next
		rows = rows[:limit]
	}
	resp := make([]discoveryRun, len(rows))
	for i, row := range rows {
		resp[i] = toDiscoveryRun(row)
	}
	h.writeJSON(w, http.StatusOK, discoveryRunPage{Runs: resp, Cursor: cursorOut})
}

func (h *Handler) handleGetDiscoveryRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.ensureDiscoveryQueries(w) {
		return
	}
	row, err := h.discovery.GetDiscoveryRun(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			h.writeError(w, http.StatusNotFound, "not_found", "discovery run not found", map[string]any{"id": id})
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "run id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("get discovery run failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to fetch discovery run", nil)
		}
		return
	}
	h.writeJSON(w, http.StatusOK, toDiscoveryRun(row))
}

func (h *Handler) handleListDiscoveryRunLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.ensureDiscoveryQueries(w) {
		return
	}
	ctx := r.Context()
	if _, err := h.discovery.GetDiscoveryRun(ctx, id); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			h.writeError(w, http.StatusNotFound, "not_found", "discovery run not found", map[string]any{"id": id})
			return
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "run id is not a valid uuid", map[string]any{"id": id})
			return
		default:
			h.log.Error().Err(err).Str("id", id).Msg("get discovery run before logs failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to fetch discovery run logs", nil)
			return
		}
	}
	limit, err := parseLimitParam(r.URL.Query().Get("limit"), 100, 500)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid limit", map[string]any{"error": err.Error()})
		return
	}
	var beforeCreatedAt *time.Time
	var beforeLogID *int64
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		ts, idValue, err := decodeCursor(cursor)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid cursor", map[string]any{"error": err.Error()})
			return
		}
		parsedID, err := strconv.ParseInt(idValue, 10, 64)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid cursor", map[string]any{"error": err.Error()})
			return
		}
		beforeCreatedAt = &ts
		beforeLogID = new(int64)
		*beforeLogID = parsedID
	}
	rows, err := h.discovery.ListDiscoveryRunLogs(ctx, sqlcgen.ListDiscoveryRunLogsParams{
		RunID:           id,
		BeforeCreatedAt: beforeCreatedAt,
		BeforeID:        beforeLogID,
		Limit:           int32(limit + 1),
	})
	if err != nil {
		h.log.Error().Err(err).Str("run_id", id).Msg("list discovery run logs failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list discovery run logs", nil)
		return
	}
	cursorOut := (*string)(nil)
	if len(rows) > limit {
		last := rows[limit-1]
		next := encodeCursor(last.CreatedAt, strconv.FormatInt(last.ID, 10))
		cursorOut = &next
		rows = rows[:limit]
	}
	resp := make([]discoveryRunLogEntry, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, toDiscoveryRunLogEntry(row))
	}
	h.writeJSON(w, http.StatusOK, discoveryRunLogPage{Logs: resp, Cursor: cursorOut})
}
