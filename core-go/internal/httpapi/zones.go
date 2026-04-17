package httpapi

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"roller_hoops/core-go/internal/sqlcgen"
)

type zone struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type zoneMember struct {
	DeviceID    string  `json:"device_id"`
	DisplayName *string `json:"display_name,omitempty"`
}

type zoneCreateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type zoneUpdateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type zoneMemberAddRequest struct {
	DeviceID string `json:"device_id"`
}

type zoneMembersReplaceRequest struct {
	DeviceIDs []string `json:"device_ids"`
}

func (h *Handler) handleListZones(w http.ResponseWriter, r *http.Request) {
	if !h.ensureTopologyQueries(w) {
		return
	}

	rows, err := h.topology.ListZones(r.Context())
	if err != nil {
		h.log.Error().Err(err).Msg("list zones failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list zones", nil)
		return
	}

	resp := make([]zone, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, toZone(row))
	}
	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleCreateZone(w http.ResponseWriter, r *http.Request) {
	if !h.ensureTopologyWriteAccess(w, r) {
		return
	}

	var req zoneCreateRequest
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}

	name, description, ok := normalizeZoneInput(req.Name, req.Description)
	if !ok {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "name is required", nil)
		return
	}

	created, err := h.topology.CreateZone(r.Context(), sqlcgen.CreateZoneParams{Name: name, Description: description})
	if err != nil {
		switch {
		case isUniqueViolation(err):
			h.writeError(w, http.StatusConflict, "conflict", "zone name already exists", map[string]any{"name": name})
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid zone payload", map[string]any{"error": err.Error()})
		default:
			h.log.Error().Err(err).Msg("create zone failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to create zone", nil)
		}
		return
	}

	if err := h.insertTopologyAudit(r, "create_zone", "zone", &created.ID, map[string]any{
		"name":        created.Name,
		"description": created.Description,
	}); err != nil {
		h.log.Error().Err(err).Msg("audit create zone failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to write audit event", nil)
		return
	}

	h.writeJSON(w, http.StatusCreated, toZone(created))
}

func (h *Handler) handleGetZone(w http.ResponseWriter, r *http.Request) {
	if !h.ensureTopologyQueries(w) {
		return
	}

	zoneID := chi.URLParam(r, "id")
	row, err := h.topology.GetZone(r.Context(), zoneID)
	if err != nil {
		h.writeZoneLookupError(w, err, zoneID)
		return
	}

	h.writeJSON(w, http.StatusOK, toZone(row))
}

func (h *Handler) handleUpdateZone(w http.ResponseWriter, r *http.Request) {
	if !h.ensureTopologyWriteAccess(w, r) {
		return
	}

	zoneID := chi.URLParam(r, "id")
	before, err := h.topology.GetZone(r.Context(), zoneID)
	if err != nil {
		h.writeZoneLookupError(w, err, zoneID)
		return
	}

	var req zoneUpdateRequest
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}

	name, description, ok := normalizeZoneInput(req.Name, req.Description)
	if !ok {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "name is required", nil)
		return
	}

	updated, err := h.topology.UpdateZone(r.Context(), sqlcgen.UpdateZoneParams{ID: zoneID, Name: name, Description: description})
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			h.writeError(w, http.StatusNotFound, "not_found", "zone not found", map[string]any{"id": zoneID})
		case isUniqueViolation(err):
			h.writeError(w, http.StatusConflict, "conflict", "zone name already exists", map[string]any{"name": name})
		case isInvalidUUID(err):
			h.writeError(w, http.StatusBadRequest, "invalid_id", "invalid zone id", map[string]any{"id": zoneID})
		default:
			h.log.Error().Err(err).Msg("update zone failed")
			h.writeError(w, http.StatusInternalServerError, "db_error", "failed to update zone", nil)
		}
		return
	}

	if err := h.insertTopologyAudit(r, "update_zone", "zone", &updated.ID, map[string]any{
		"before": toZone(before),
		"after":  toZone(updated),
	}); err != nil {
		h.log.Error().Err(err).Msg("audit update zone failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to write audit event", nil)
		return
	}

	h.writeJSON(w, http.StatusOK, toZone(updated))
}

func (h *Handler) handleDeleteZone(w http.ResponseWriter, r *http.Request) {
	if !h.ensureTopologyWriteAccess(w, r) {
		return
	}

	zoneID := chi.URLParam(r, "id")
	before, err := h.topology.GetZone(r.Context(), zoneID)
	if err != nil {
		h.writeZoneLookupError(w, err, zoneID)
		return
	}

	deleted, err := h.topology.DeleteZone(r.Context(), zoneID)
	if err != nil {
		if isInvalidUUID(err) {
			h.writeError(w, http.StatusBadRequest, "invalid_id", "invalid zone id", map[string]any{"id": zoneID})
			return
		}
		h.log.Error().Err(err).Msg("delete zone failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to delete zone", nil)
		return
	}
	if deleted == 0 {
		h.writeError(w, http.StatusNotFound, "not_found", "zone not found", map[string]any{"id": zoneID})
		return
	}

	if err := h.insertTopologyAudit(r, "delete_zone", "zone", &zoneID, map[string]any{"before": toZone(before)}); err != nil {
		h.log.Error().Err(err).Msg("audit delete zone failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to write audit event", nil)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleListZoneMembers(w http.ResponseWriter, r *http.Request) {
	if !h.ensureTopologyQueries(w) {
		return
	}

	zoneID := chi.URLParam(r, "id")
	if _, err := h.topology.GetZone(r.Context(), zoneID); err != nil {
		h.writeZoneLookupError(w, err, zoneID)
		return
	}

	rows, err := h.topology.ListZoneMembers(r.Context(), zoneID)
	if err != nil {
		if isInvalidUUID(err) {
			h.writeError(w, http.StatusBadRequest, "invalid_id", "invalid zone id", map[string]any{"id": zoneID})
			return
		}
		h.log.Error().Err(err).Msg("list zone members failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list zone members", nil)
		return
	}

	resp := make([]zoneMember, 0, len(rows))
	for _, row := range rows {
		resp = append(resp, toZoneMember(row))
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleReplaceZoneMembers(w http.ResponseWriter, r *http.Request) {
	if !h.ensureTopologyWriteAccess(w, r) {
		return
	}

	zoneID := chi.URLParam(r, "id")
	if _, err := h.topology.GetZone(r.Context(), zoneID); err != nil {
		h.writeZoneLookupError(w, err, zoneID)
		return
	}

	var req zoneMembersReplaceRequest
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}

	deviceIDs, ok := normalizeDeviceIDList(req.DeviceIDs)
	if !ok {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "device_ids cannot contain blank values", nil)
		return
	}
	if err := h.ensureDevicesExist(r, deviceIDs); err != nil {
		h.writeZoneMembershipValidationError(w, err)
		return
	}

	before, err := h.topology.ListZoneMembers(r.Context(), zoneID)
	if err != nil {
		h.log.Error().Err(err).Msg("list zone members before replace failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to list zone members", nil)
		return
	}

	if err := h.topology.ReplaceZoneMembers(r.Context(), sqlcgen.ReplaceZoneMembersParams{ZoneID: zoneID, DeviceIDs: deviceIDs}); err != nil {
		if isInvalidUUID(err) {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid device_ids payload", map[string]any{"error": err.Error()})
			return
		}
		if isForeignKeyViolation(err) {
			h.writeError(w, http.StatusNotFound, "not_found", "device not found", nil)
			return
		}
		h.log.Error().Err(err).Msg("replace zone members failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to replace zone members", nil)
		return
	}

	if err := h.insertTopologyAudit(r, "set_zone_members", "zone", &zoneID, map[string]any{
		"before":         toZoneMembers(before),
		"after_device_ids": deviceIDs,
	}); err != nil {
		h.log.Error().Err(err).Msg("audit replace zone members failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to write audit event", nil)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleAddZoneMember(w http.ResponseWriter, r *http.Request) {
	if !h.ensureTopologyWriteAccess(w, r) {
		return
	}

	zoneID := chi.URLParam(r, "id")
	if _, err := h.topology.GetZone(r.Context(), zoneID); err != nil {
		h.writeZoneLookupError(w, err, zoneID)
		return
	}

	var req zoneMemberAddRequest
	if err := decodeJSONStrict(r, &req); err != nil {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid json body", map[string]any{"error": err.Error()})
		return
	}

	deviceID := strings.TrimSpace(req.DeviceID)
	if deviceID == "" {
		h.writeError(w, http.StatusBadRequest, "validation_failed", "device_id is required", nil)
		return
	}
	if err := h.ensureDevicesExist(r, []string{deviceID}); err != nil {
		h.writeZoneMembershipValidationError(w, err)
		return
	}

	added, err := h.topology.AddZoneMember(r.Context(), sqlcgen.AddZoneMemberParams{ZoneID: zoneID, DeviceID: deviceID})
	if err != nil {
		if isInvalidUUID(err) {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid zone or device id", map[string]any{"zone_id": zoneID, "device_id": deviceID})
			return
		}
		if isForeignKeyViolation(err) {
			h.writeError(w, http.StatusNotFound, "not_found", "device not found", map[string]any{"device_id": deviceID})
			return
		}
		h.log.Error().Err(err).Msg("add zone member failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to add zone member", nil)
		return
	}

	if err := h.insertTopologyAudit(r, "add_zone_member", "zone", &zoneID, map[string]any{
		"device_id": deviceID,
		"created":   added > 0,
	}); err != nil {
		h.log.Error().Err(err).Msg("audit add zone member failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to write audit event", nil)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) handleRemoveZoneMember(w http.ResponseWriter, r *http.Request) {
	if !h.ensureTopologyWriteAccess(w, r) {
		return
	}

	zoneID := chi.URLParam(r, "id")
	deviceID := chi.URLParam(r, "deviceId")
	if _, err := h.topology.GetZone(r.Context(), zoneID); err != nil {
		h.writeZoneLookupError(w, err, zoneID)
		return
	}

	removed, err := h.topology.RemoveZoneMember(r.Context(), sqlcgen.RemoveZoneMemberParams{ZoneID: zoneID, DeviceID: deviceID})
	if err != nil {
		if isInvalidUUID(err) {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid zone or device id", map[string]any{"zone_id": zoneID, "device_id": deviceID})
			return
		}
		h.log.Error().Err(err).Msg("remove zone member failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to remove zone member", nil)
		return
	}
	if removed == 0 {
		h.writeError(w, http.StatusNotFound, "not_found", "zone membership not found", map[string]any{"zone_id": zoneID, "device_id": deviceID})
		return
	}

	if err := h.insertTopologyAudit(r, "remove_zone_member", "zone", &zoneID, map[string]any{"device_id": deviceID}); err != nil {
		h.log.Error().Err(err).Msg("audit remove zone member failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to write audit event", nil)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) ensureTopologyWriteAccess(w http.ResponseWriter, r *http.Request) bool {
	if !h.ensureTopologyQueries(w) || !h.ensureAuditQueries(w) {
		return false
	}
	if strings.TrimSpace(r.Header.Get("X-User-Role")) != "admin" {
		h.writeError(w, http.StatusForbidden, "forbidden", "admin role required", nil)
		return false
	}
	return true
}

func (h *Handler) insertTopologyAudit(r *http.Request, action, targetType string, targetID *string, details map[string]any) error {
	actor := strings.TrimSpace(r.Header.Get("X-User"))
	if actor == "" {
		actor = "unknown"
	}
	actorRole := strings.TrimSpace(r.Header.Get("X-User-Role"))
	if actorRole == "" {
		actorRole = "unknown"
	}
	return h.audit.InsertAuditEvent(r.Context(), sqlcgen.InsertAuditEventParams{
		Actor:      actor,
		ActorRole:  &actorRole,
		Action:     action,
		TargetType: &targetType,
		TargetID:   targetID,
		Details:    details,
	})
}

func (h *Handler) ensureDevicesExist(r *http.Request, deviceIDs []string) error {
	if len(deviceIDs) == 0 {
		return nil
	}
	existing, err := h.topology.ListExistingDeviceIDs(r.Context(), deviceIDs)
	if err != nil {
		return err
	}
	missing := make([]string, 0)
	for _, id := range deviceIDs {
		if !slices.Contains(existing, id) {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return errMissingDevices{IDs: missing}
	}
	return nil
}

type errMissingDevices struct {
	IDs []string
}

func (e errMissingDevices) Error() string {
	return "one or more devices not found"
}

func (h *Handler) writeZoneLookupError(w http.ResponseWriter, err error, zoneID string) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		h.writeError(w, http.StatusNotFound, "not_found", "zone not found", map[string]any{"id": zoneID})
	case isInvalidUUID(err):
		h.writeError(w, http.StatusBadRequest, "invalid_id", "invalid zone id", map[string]any{"id": zoneID})
	default:
		h.log.Error().Err(err).Msg("zone lookup failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to load zone", nil)
	}
}

func (h *Handler) writeZoneMembershipValidationError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case errMissingDevices:
		h.writeError(w, http.StatusNotFound, "not_found", "device not found", map[string]any{"device_ids": e.IDs})
	case *pgconn.PgError:
		if isInvalidUUID(err) {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid device id", map[string]any{"error": err.Error()})
			return
		}
		h.log.Error().Err(err).Msg("zone membership validation failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to validate devices", nil)
	default:
		if isInvalidUUID(err) {
			h.writeError(w, http.StatusBadRequest, "validation_failed", "invalid device id", map[string]any{"error": err.Error()})
			return
		}
		h.log.Error().Err(err).Msg("zone membership validation failed")
		h.writeError(w, http.StatusInternalServerError, "db_error", "failed to validate devices", nil)
	}
}

func normalizeZoneInput(name string, description *string) (string, *string, bool) {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return "", nil, false
	}
	return trimmedName, normalizeOptionalString(description), true
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeDeviceIDList(values []string) ([]string, bool) {
	if len(values) == 0 {
		return []string{}, true
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return nil, false
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result, true
}

func toZone(v sqlcgen.Zone) zone {
	return zone{
		ID:          v.ID,
		Name:        v.Name,
		Description: v.Description,
		CreatedAt:   v.CreatedAt,
		UpdatedAt:   v.UpdatedAt,
	}
}

func toZoneMember(v sqlcgen.ZoneMember) zoneMember {
	return zoneMember{DeviceID: v.DeviceID, DisplayName: v.DisplayName}
}

func toZoneMembers(values []sqlcgen.ZoneMember) []zoneMember {
	out := make([]zoneMember, 0, len(values))
	for _, value := range values {
		out = append(out, toZoneMember(value))
	}
	return out
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
