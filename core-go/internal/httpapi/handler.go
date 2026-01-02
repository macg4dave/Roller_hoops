package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog"

	"roller_hoops/core-go/internal/db"
	"roller_hoops/core-go/internal/metrics"
	"roller_hoops/core-go/internal/sqlcgen"
)

type Handler struct {
	log       zerolog.Logger
	pool      *db.Pool
	devices   deviceQueries
	discovery discoveryQueries
	inventory inventoryQueries
	audit     auditQueries
	metrics   *metrics.Metrics
}

type deviceQueries interface {
	ListDevices(ctx context.Context) ([]sqlcgen.Device, error)
	ListDevicesPage(ctx context.Context, arg sqlcgen.ListDevicesPageParams) ([]sqlcgen.DeviceListItem, error)
	GetDevice(ctx context.Context, id string) (sqlcgen.Device, error)
	CreateDevice(ctx context.Context, displayName *string) (sqlcgen.Device, error)
	UpdateDevice(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error)
	UpsertDeviceMetadata(ctx context.Context, arg sqlcgen.UpsertDeviceMetadataParams) (sqlcgen.DeviceMetadata, error)
	ListDeviceNameCandidates(ctx context.Context, deviceID string) ([]sqlcgen.DeviceNameCandidate, error)
	ListDeviceIPs(ctx context.Context, deviceID string) ([]sqlcgen.DeviceIP, error)
	ListDeviceMACs(ctx context.Context, deviceID string) ([]sqlcgen.DeviceMAC, error)
	ListDeviceInterfaces(ctx context.Context, deviceID string) ([]sqlcgen.DeviceInterface, error)
	ListDeviceServices(ctx context.Context, deviceID string) ([]sqlcgen.DeviceService, error)
	GetDeviceSNMP(ctx context.Context, deviceID string) (sqlcgen.DeviceSNMP, error)
	ListDeviceLinks(ctx context.Context, deviceID string) ([]sqlcgen.DeviceLink, error)
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

type auditQueries interface {
	InsertAuditEvent(ctx context.Context, arg sqlcgen.InsertAuditEventParams) error
}

func NewHandler(log zerolog.Logger, pool *db.Pool) *Handler {
	return newHandler(log, pool, nil)
}

func NewHandlerWithMetrics(log zerolog.Logger, pool *db.Pool, m *metrics.Metrics) *Handler {
	return newHandler(log, pool, m)
}

func newHandler(log zerolog.Logger, pool *db.Pool, m *metrics.Metrics) *Handler {
	var dq deviceQueries
	var drq discoveryQueries
	var iq inventoryQueries
	var aq auditQueries
	if pool != nil {
		q := pool.Queries()
		dq = q
		drq = q
		iq = q
		aq = q
	}
	return &Handler{log: log, pool: pool, devices: dq, discovery: drq, inventory: iq, audit: aq, metrics: m}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(h.ensureResponseRequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(h.accessLog)

	if h.metrics != nil {
		r.Handle("/metrics", h.metrics.Handler())
	}

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
					r.Get("/facts", h.handleGetDeviceFacts)
					r.Get("/name-candidates", h.handleListDeviceNameCandidates)
					r.Get("/history", h.handleDeviceHistory)
					r.Put("/", h.handleUpdateDevice)
				})
			})

			r.Route("/discovery", func(r chi.Router) {
				r.Post("/run", h.handleDiscoveryRun)
				r.Get("/status", h.handleDiscoveryStatus)
				r.Get("/scope-suggestions", h.handleDiscoveryScopeSuggestions)
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

			r.Route("/audit", func(r chi.Router) {
				r.Post("/events", h.handleCreateAuditEvent)
			})

			r.Route("/map", func(r chi.Router) {
				r.Get("/{layer}", h.handleGetMapProjection)
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

		status := ww.Status()
		duration := time.Since(start)
		h.log.Info().
			Str("request_id", middleware.GetReqID(r.Context())).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", status).
			Int("bytes", ww.BytesWritten()).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("http_request")

		h.observeRequestMetrics(r, status, duration)
	})
}

func (h *Handler) observeRequestMetrics(r *http.Request, status int, duration time.Duration) {
	if h.metrics == nil {
		return
	}
	h.metrics.ObserveHTTPRequest(r.Method, requestRouteLabel(r), status, duration)
}

func requestRouteLabel(r *http.Request) string {
	if rc := chi.RouteContext(r.Context()); rc != nil {
		if pattern := rc.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return r.URL.Path
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
	ID           string          `json:"id"`
	DisplayName  *string         `json:"display_name,omitempty"`
	PrimaryIP    *string         `json:"primary_ip,omitempty"`
	Metadata     *deviceMetadata `json:"metadata,omitempty"`
	LastSeenAt   *time.Time      `json:"last_seen_at,omitempty"`
	LastChangeAt *time.Time      `json:"last_change_at,omitempty"`
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

type devicePage struct {
	Devices []device `json:"devices"`
	Cursor  *string  `json:"cursor,omitempty"`
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

type deviceIPFact struct {
	IP            string    `json:"ip"`
	InterfaceID   *string   `json:"interface_id,omitempty"`
	InterfaceName *string   `json:"interface_name,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type deviceMACFact struct {
	MAC           string    `json:"mac"`
	InterfaceID   *string   `json:"interface_id,omitempty"`
	InterfaceName *string   `json:"interface_name,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type deviceInterfaceFact struct {
	ID             string     `json:"id"`
	Name           *string    `json:"name,omitempty"`
	Ifindex        *int32     `json:"ifindex,omitempty"`
	Descr          *string    `json:"descr,omitempty"`
	Alias          *string    `json:"alias,omitempty"`
	MAC            *string    `json:"mac,omitempty"`
	AdminStatus    *int32     `json:"admin_status,omitempty"`
	OperStatus     *int32     `json:"oper_status,omitempty"`
	MTU            *int32     `json:"mtu,omitempty"`
	SpeedBps       *int64     `json:"speed_bps,omitempty"`
	PVID           *int32     `json:"pvid,omitempty"`
	PVIDObservedAt *time.Time `json:"pvid_observed_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

type deviceServiceFact struct {
	Protocol   *string   `json:"protocol,omitempty"`
	Port       *int32    `json:"port,omitempty"`
	Name       *string   `json:"name,omitempty"`
	State      *string   `json:"state,omitempty"`
	Source     *string   `json:"source,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type deviceSNMPFact struct {
	Address       *string    `json:"address,omitempty"`
	SysName       *string    `json:"sys_name,omitempty"`
	SysDescr      *string    `json:"sys_descr,omitempty"`
	SysObjectID   *string    `json:"sys_object_id,omitempty"`
	SysContact    *string    `json:"sys_contact,omitempty"`
	SysLocation   *string    `json:"sys_location,omitempty"`
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`
	LastError     *string    `json:"last_error,omitempty"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type deviceLinkFact struct {
	ID               string     `json:"id"`
	LinkKey          string     `json:"link_key"`
	PeerDeviceID     string     `json:"peer_device_id"`
	LocalInterfaceID *string    `json:"local_interface_id,omitempty"`
	PeerInterfaceID  *string    `json:"peer_interface_id,omitempty"`
	LinkType         *string    `json:"link_type,omitempty"`
	Source           string     `json:"source"`
	ObservedAt       *time.Time `json:"observed_at,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type deviceFacts struct {
	DeviceID   string                `json:"device_id"`
	IPs        []deviceIPFact        `json:"ips"`
	MACs       []deviceMACFact       `json:"macs"`
	Interfaces []deviceInterfaceFact `json:"interfaces"`
	Services   []deviceServiceFact   `json:"services"`
	SNMP       *deviceSNMPFact       `json:"snmp,omitempty"`
	Links      []deviceLinkFact      `json:"links"`
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

type auditEventCreate struct {
	Actor      string         `json:"actor"`
	ActorRole  *string        `json:"actor_role,omitempty"`
	Action     string         `json:"action"`
	TargetType *string        `json:"target_type,omitempty"`
	TargetID   *string        `json:"target_id,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
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

func (h *Handler) ensureAuditQueries(w http.ResponseWriter) bool {
	if h.audit == nil {
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

func toDeviceListItem(d sqlcgen.DeviceListItem) device {
	var meta *deviceMetadata
	if d.Owner != nil || d.Location != nil || d.Notes != nil {
		meta = &deviceMetadata{
			Owner:    d.Owner,
			Location: d.Location,
			Notes:    d.Notes,
		}
	}
	lastChangeAt := d.LastChangeAt
	return device{
		ID:           d.ID,
		DisplayName:  d.DisplayName,
		PrimaryIP:    d.PrimaryIP,
		Metadata:     meta,
		LastSeenAt:   d.LastSeenAt,
		LastChangeAt: &lastChangeAt,
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
	Scope  *string  `json:"scope,omitempty"`
	Preset *string  `json:"preset,omitempty"`
	Tags   []string `json:"tags,omitempty"`
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

func normalizeScanPreset(preset *string) *string {
	if preset == nil {
		return nil
	}
	s := strings.ToLower(strings.TrimSpace(*preset))
	if s == "" {
		return nil
	}
	return &s
}

func validateScanPreset(preset *string) (*string, error) {
	preset = normalizeScanPreset(preset)
	if preset == nil {
		defaultPreset := "normal"
		return &defaultPreset, nil
	}

	switch *preset {
	case "fast", "normal", "deep":
		return preset, nil
	default:
		return nil, fmt.Errorf("preset must be one of: fast, normal, deep")
	}
}

var allowedScanTags = map[string]struct{}{
	"ports":    {},
	"snmp":     {},
	"topology": {},
	"names":    {},
}

func normalizeScanTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		s := strings.ToLower(strings.TrimSpace(raw))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func validateScanTags(tags []string) ([]string, error) {
	tags = normalizeScanTags(tags)
	if len(tags) == 0 {
		return nil, nil
	}
	if len(tags) > 8 {
		return nil, fmt.Errorf("too many tags (max 8)")
	}
	for _, tag := range tags {
		if _, ok := allowedScanTags[tag]; !ok {
			return nil, fmt.Errorf("unknown tag %q", tag)
		}
	}
	sort.Strings(tags)
	return tags, nil
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

func parsePositiveIntParam(value string, defaultVal, max int) (int, error) {
	if value == "" {
		return defaultVal, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid value")
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("must be positive")
	}
	if parsed > max {
		parsed = max
	}
	return parsed, nil
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

	queryParam := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	sort := strings.TrimSpace(r.URL.Query().Get("sort"))
	if sort == "" {
		sort = "created_desc"
	}
	switch sort {
	case "created_desc", "last_seen_desc", "last_change_desc":
	default:
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid sort value", map[string]any{"sort": sort})
		return
	}
	if status != "" {
		switch status {
		case "online", "offline", "changed":
		default:
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid status value", map[string]any{"status": status})
			return
		}
	}

	limit, err := parseLimitParam(r.URL.Query().Get("limit"), 50, 200)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid limit", map[string]any{"error": err.Error()})
		return
	}

	seenWithinSeconds, err := parsePositiveIntParam(r.URL.Query().Get("seen_within_seconds"), 3600, 30*24*3600)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid seen_within_seconds", map[string]any{"error": err.Error()})
		return
	}
	changedWithinSeconds, err := parsePositiveIntParam(r.URL.Query().Get("changed_within_seconds"), 86400, 30*24*3600)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid changed_within_seconds", map[string]any{"error": err.Error()})
		return
	}
	now := time.Now().UTC()
	seenAfter := now.Add(-time.Duration(seenWithinSeconds) * time.Second)
	changedAfter := now.Add(-time.Duration(changedWithinSeconds) * time.Second)

	var beforeSortTs *time.Time
	var beforeID *string
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		ts, id, err := decodeCursor(cursor)
		if err != nil {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid cursor", map[string]any{"error": err.Error()})
			return
		}
		beforeSortTs = &ts
		beforeID = &id
	}

	var queryLike *string
	if queryParam != "" {
		pat := "%" + queryParam + "%"
		queryLike = &pat
	}
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	rows, err := h.devices.ListDevicesPage(r.Context(), sqlcgen.ListDevicesPageParams{
		Query:        queryLike,
		Status:       statusPtr,
		Sort:         sort,
		SeenAfter:    seenAfter,
		ChangedAfter: changedAfter,
		BeforeSortTs: beforeSortTs,
		BeforeID:     beforeID,
		Limit:        int32(limit + 1),
	})
	if err != nil {
		h.log.Error().Err(err).Msg("list devices failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list devices", nil)
		return
	}

	cursorOut := (*string)(nil)
	if len(rows) > limit {
		last := rows[limit-1]
		next := encodeCursor(last.SortTs, last.ID)
		cursorOut = &next
		rows = rows[:limit]
	}
	resp := make([]device, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, toDeviceListItem(row))
	}
	h.writeJSON(w, http.StatusOK, devicePage{Devices: resp, Cursor: cursorOut})
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

func (h *Handler) handleCreateAuditEvent(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAuditQueries(w) {
		return
	}
	var req auditEventCreate
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}
	req.Actor = strings.TrimSpace(req.Actor)
	req.Action = strings.TrimSpace(req.Action)
	if req.Actor == "" || req.Action == "" {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "actor and action are required", nil)
		return
	}
	if req.TargetType != nil {
		if s := strings.TrimSpace(*req.TargetType); s == "" {
			req.TargetType = nil
		} else {
			req.TargetType = &s
		}
	}
	if req.TargetID != nil {
		if s := strings.TrimSpace(*req.TargetID); s == "" {
			req.TargetID = nil
		} else {
			req.TargetID = &s
		}
	}
	if req.ActorRole != nil {
		if s := strings.TrimSpace(*req.ActorRole); s == "" {
			req.ActorRole = nil
		} else {
			req.ActorRole = &s
		}
	}
	details := req.Details
	if details == nil {
		details = map[string]any{}
	}

	if err := h.audit.InsertAuditEvent(r.Context(), sqlcgen.InsertAuditEventParams{
		Actor:      req.Actor,
		ActorRole:  req.ActorRole,
		Action:     req.Action,
		TargetType: req.TargetType,
		TargetID:   req.TargetID,
		Details:    details,
	}); err != nil {
		if isInvalidUUID(err) {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid target_id", map[string]any{"target_id": req.TargetID})
			return
		}
		h.log.Error().Err(err).Msg("insert audit event failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to write audit event", nil)
		return
	}

	h.writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
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

func (h *Handler) handleGetDeviceFacts(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !h.ensureDeviceQueries(w) {
		return
	}

	ctx := r.Context()
	if _, err := h.devices.GetDevice(ctx, id); err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			h.writeError(w, http.StatusNotFound, "not_found", "device not found", map[string]any{"id": id})
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("fetch device before facts failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to fetch device facts", nil)
		}
		return
	}

	ips, err := h.devices.ListDeviceIPs(ctx, id)
	if err != nil {
		switch {
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("list device ips failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list device ips", nil)
		}
		return
	}
	macs, err := h.devices.ListDeviceMACs(ctx, id)
	if err != nil {
		switch {
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("list device macs failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list device macs", nil)
		}
		return
	}
	ifaces, err := h.devices.ListDeviceInterfaces(ctx, id)
	if err != nil {
		switch {
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("list device interfaces failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list device interfaces", nil)
		}
		return
	}
	services, err := h.devices.ListDeviceServices(ctx, id)
	if err != nil {
		switch {
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("list device services failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list device services", nil)
		}
		return
	}
	links, err := h.devices.ListDeviceLinks(ctx, id)
	if err != nil {
		switch {
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("list device links failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list device links", nil)
		}
		return
	}

	var snmpOut *deviceSNMPFact
	if snmpRow, err := h.devices.GetDeviceSNMP(ctx, id); err == nil {
		snmpOut = &deviceSNMPFact{
			Address:       snmpRow.Address,
			SysName:       snmpRow.SysName,
			SysDescr:      snmpRow.SysDescr,
			SysObjectID:   snmpRow.SysObjectID,
			SysContact:    snmpRow.SysContact,
			SysLocation:   snmpRow.SysLocation,
			LastSuccessAt: snmpRow.LastSuccessAt,
			LastError:     snmpRow.LastError,
			UpdatedAt:     snmpRow.UpdatedAt,
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		switch {
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "device id is not a valid uuid", map[string]any{"id": id})
		default:
			h.log.Error().Err(err).Str("id", id).Msg("get device snmp failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to fetch device snmp facts", nil)
		}
		return
	}

	ipFacts := make([]deviceIPFact, 0, len(ips))
	for _, row := range ips {
		ipFacts = append(ipFacts, deviceIPFact{
			IP:            row.IP,
			InterfaceID:   row.InterfaceID,
			InterfaceName: row.InterfaceName,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		})
	}
	macFacts := make([]deviceMACFact, 0, len(macs))
	for _, row := range macs {
		macFacts = append(macFacts, deviceMACFact{
			MAC:           row.MAC,
			InterfaceID:   row.InterfaceID,
			InterfaceName: row.InterfaceName,
			CreatedAt:     row.CreatedAt,
			UpdatedAt:     row.UpdatedAt,
		})
	}
	ifaceFacts := make([]deviceInterfaceFact, 0, len(ifaces))
	for _, row := range ifaces {
		ifaceFacts = append(ifaceFacts, deviceInterfaceFact{
			ID:             row.ID,
			Name:           row.Name,
			Ifindex:        row.Ifindex,
			Descr:          row.Descr,
			Alias:          row.Alias,
			MAC:            row.MAC,
			AdminStatus:    row.AdminStatus,
			OperStatus:     row.OperStatus,
			MTU:            row.MTU,
			SpeedBps:       row.SpeedBps,
			PVID:           row.PVID,
			PVIDObservedAt: row.PVIDObservedAt,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		})
	}
	serviceFacts := make([]deviceServiceFact, 0, len(services))
	for _, row := range services {
		serviceFacts = append(serviceFacts, deviceServiceFact{
			Protocol:   row.Protocol,
			Port:       row.Port,
			Name:       row.Name,
			State:      row.State,
			Source:     row.Source,
			ObservedAt: row.ObservedAt,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		})
	}
	linkFacts := make([]deviceLinkFact, 0, len(links))
	for _, row := range links {
		linkFacts = append(linkFacts, deviceLinkFact{
			ID:               row.ID,
			LinkKey:          row.LinkKey,
			PeerDeviceID:     row.PeerDeviceID,
			LocalInterfaceID: row.LocalInterfaceID,
			PeerInterfaceID:  row.PeerInterfaceID,
			LinkType:         row.LinkType,
			Source:           row.Source,
			ObservedAt:       row.ObservedAt,
			UpdatedAt:        row.UpdatedAt,
		})
	}

	h.writeJSON(w, http.StatusOK, deviceFacts{
		DeviceID:   id,
		IPs:        ipFacts,
		MACs:       macFacts,
		Interfaces: ifaceFacts,
		Services:   serviceFacts,
		SNMP:       snmpOut,
		Links:      linkFacts,
	})
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

	req.Preset, err = validateScanPreset(req.Preset)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid scan preset", map[string]any{"error": err.Error()})
		return
	}

	req.Tags, err = validateScanTags(req.Tags)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid scan tags", map[string]any{"error": err.Error()})
		return
	}

	stats := map[string]any{"stage": "queued", "preset": *req.Preset}
	if len(req.Tags) > 0 {
		stats["tags"] = req.Tags
	}

	run, err := h.discovery.InsertDiscoveryRun(r.Context(), sqlcgen.InsertDiscoveryRunParams{
		Status: "queued",
		Scope:  req.Scope,
		Stats:  stats,
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

type discoveryScopeSuggestion struct {
	Scope     string  `json:"scope"`
	Interface *string `json:"interface,omitempty"`
	Address   *string `json:"address,omitempty"`
}

type discoveryScopeSuggestionsResponse struct {
	Scopes []discoveryScopeSuggestion `json:"scopes"`
}

func (h *Handler) handleDiscoveryScopeSuggestions(w http.ResponseWriter, r *http.Request) {
	suggestions := buildDiscoveryScopeSuggestions()
	h.writeJSON(w, http.StatusOK, discoveryScopeSuggestionsResponse{Scopes: suggestions})
}

func buildDiscoveryScopeSuggestions() []discoveryScopeSuggestion {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	seen := make(map[string]discoveryScopeSuggestion)
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		ifaceName := strings.TrimSpace(iface.Name)
		if ifaceName == "" {
			ifaceName = "interface"
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet == nil {
				continue
			}
			ip := ipNet.IP
			if ip == nil {
				continue
			}

			netIP := ip
			if v4 := ip.To4(); v4 != nil {
				netIP = v4
			}
			netipAddr, ok := netip.AddrFromSlice(netIP)
			if !ok {
				continue
			}
			netipAddr = netipAddr.Unmap()
			if netipAddr.IsLoopback() || netipAddr.IsUnspecified() || netipAddr.IsLinkLocalUnicast() {
				continue
			}

			ones, bits := ipNet.Mask.Size()
			if ones <= 0 || bits <= 0 {
				continue
			}
			prefix := netip.PrefixFrom(netipAddr, ones).Masked()
			scope := prefix.String()

			if scope == "" {
				continue
			}

			if _, exists := seen[scope]; exists {
				continue
			}

			ifaceCopy := ifaceName
			ipStr := netipAddr.String()
			ipCopy := ipStr
			seen[scope] = discoveryScopeSuggestion{
				Scope:     scope,
				Interface: &ifaceCopy,
				Address:   &ipCopy,
			}
		}
	}

	out := make([]discoveryScopeSuggestion, 0, len(seen))
	for _, entry := range seen {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Scope < out[j].Scope
	})
	return out
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
