package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"roller_hoops/core-go/internal/metrics"
	"roller_hoops/core-go/internal/sqlcgen"
)

type fakeDeviceQueries struct {
	listFn               func(ctx context.Context) ([]sqlcgen.Device, error)
	listPageFn           func(ctx context.Context, arg sqlcgen.ListDevicesPageParams) ([]sqlcgen.DeviceListItem, error)
	getFn                func(ctx context.Context, id string) (sqlcgen.Device, error)
	createFn             func(ctx context.Context, displayName *string) (sqlcgen.Device, error)
	updateFn             func(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error)
	upsertFn             func(ctx context.Context, arg sqlcgen.UpsertDeviceMetadataParams) (sqlcgen.DeviceMetadata, error)
	listTagsFn           func(ctx context.Context, deviceID string) ([]sqlcgen.DeviceTag, error)
	listEffectiveTagsFn  func(ctx context.Context, deviceID string) ([]string, error)
	deleteTagsBySourceFn func(ctx context.Context, arg sqlcgen.DeleteDeviceTagsBySourceParams) error
	upsertTagFn          func(ctx context.Context, arg sqlcgen.UpsertDeviceTagParams) error
	listNameCandidatesFn func(ctx context.Context, deviceID string) ([]sqlcgen.DeviceNameCandidate, error)
	listIPsFn            func(ctx context.Context, deviceID string) ([]sqlcgen.DeviceIP, error)
	listMACsFn           func(ctx context.Context, deviceID string) ([]sqlcgen.DeviceMAC, error)
	listInterfacesFn     func(ctx context.Context, deviceID string) ([]sqlcgen.DeviceInterface, error)
	listServicesFn       func(ctx context.Context, deviceID string) ([]sqlcgen.DeviceService, error)
	getSNMPFn            func(ctx context.Context, deviceID string) (sqlcgen.DeviceSNMP, error)
	listLinksFn          func(ctx context.Context, deviceID string) ([]sqlcgen.DeviceLink, error)
	listChangeEventsFn   func(ctx context.Context, arg sqlcgen.ListDeviceChangeEventsParams) ([]sqlcgen.DeviceChangeEvent, error)
	listHistoryFn        func(ctx context.Context, arg sqlcgen.ListDeviceChangeEventsForDeviceParams) ([]sqlcgen.DeviceChangeEvent, error)
}

func (f fakeDeviceQueries) ListDevices(ctx context.Context) ([]sqlcgen.Device, error) {
	return f.listFn(ctx)
}

func (f fakeDeviceQueries) ListDevicesPage(ctx context.Context, arg sqlcgen.ListDevicesPageParams) ([]sqlcgen.DeviceListItem, error) {
	if f.listPageFn != nil {
		return f.listPageFn(ctx, arg)
	}
	if f.listFn == nil {
		return nil, nil
	}
	rows, err := f.listFn(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]sqlcgen.DeviceListItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, sqlcgen.DeviceListItem{
			ID:           row.ID,
			DisplayName:  row.DisplayName,
			Owner:        row.Owner,
			Location:     row.Location,
			Notes:        row.Notes,
			CreatedAt:    now,
			UpdatedAt:    now,
			LastSeenAt:   nil,
			LastChangeAt: now,
			SortTs:       now,
		})
	}
	return out, nil
}

func (f fakeDeviceQueries) GetDevice(ctx context.Context, id string) (sqlcgen.Device, error) {
	return f.getFn(ctx, id)
}

func (f fakeDeviceQueries) CreateDevice(ctx context.Context, displayName *string) (sqlcgen.Device, error) {
	return f.createFn(ctx, displayName)
}

func (f fakeDeviceQueries) UpdateDevice(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error) {
	return f.updateFn(ctx, arg)
}

func (f fakeDeviceQueries) UpsertDeviceMetadata(ctx context.Context, arg sqlcgen.UpsertDeviceMetadataParams) (sqlcgen.DeviceMetadata, error) {
	if f.upsertFn == nil {
		return sqlcgen.DeviceMetadata{}, nil
	}
	return f.upsertFn(ctx, arg)
}

func (f fakeDeviceQueries) ListDeviceTags(ctx context.Context, deviceID string) ([]sqlcgen.DeviceTag, error) {
	if f.listTagsFn == nil {
		return nil, nil
	}
	return f.listTagsFn(ctx, deviceID)
}

func (f fakeDeviceQueries) ListDeviceEffectiveTags(ctx context.Context, deviceID string) ([]string, error) {
	if f.listEffectiveTagsFn == nil {
		return nil, nil
	}
	return f.listEffectiveTagsFn(ctx, deviceID)
}

func (f fakeDeviceQueries) DeleteDeviceTagsBySource(ctx context.Context, arg sqlcgen.DeleteDeviceTagsBySourceParams) error {
	if f.deleteTagsBySourceFn == nil {
		return nil
	}
	return f.deleteTagsBySourceFn(ctx, arg)
}

func (f fakeDeviceQueries) UpsertDeviceTag(ctx context.Context, arg sqlcgen.UpsertDeviceTagParams) error {
	if f.upsertTagFn == nil {
		return nil
	}
	return f.upsertTagFn(ctx, arg)
}

func (f fakeDeviceQueries) ListDeviceNameCandidates(ctx context.Context, deviceID string) ([]sqlcgen.DeviceNameCandidate, error) {
	if f.listNameCandidatesFn == nil {
		return nil, nil
	}
	return f.listNameCandidatesFn(ctx, deviceID)
}

func (f fakeDeviceQueries) ListDeviceIPs(ctx context.Context, deviceID string) ([]sqlcgen.DeviceIP, error) {
	if f.listIPsFn == nil {
		return nil, nil
	}
	return f.listIPsFn(ctx, deviceID)
}

func (f fakeDeviceQueries) ListDeviceMACs(ctx context.Context, deviceID string) ([]sqlcgen.DeviceMAC, error) {
	if f.listMACsFn == nil {
		return nil, nil
	}
	return f.listMACsFn(ctx, deviceID)
}

func (f fakeDeviceQueries) ListDeviceInterfaces(ctx context.Context, deviceID string) ([]sqlcgen.DeviceInterface, error) {
	if f.listInterfacesFn == nil {
		return nil, nil
	}
	return f.listInterfacesFn(ctx, deviceID)
}

func (f fakeDeviceQueries) ListDeviceServices(ctx context.Context, deviceID string) ([]sqlcgen.DeviceService, error) {
	if f.listServicesFn == nil {
		return nil, nil
	}
	return f.listServicesFn(ctx, deviceID)
}

func (f fakeDeviceQueries) GetDeviceSNMP(ctx context.Context, deviceID string) (sqlcgen.DeviceSNMP, error) {
	if f.getSNMPFn == nil {
		return sqlcgen.DeviceSNMP{}, pgx.ErrNoRows
	}
	return f.getSNMPFn(ctx, deviceID)
}

func (f fakeDeviceQueries) ListDeviceLinks(ctx context.Context, deviceID string) ([]sqlcgen.DeviceLink, error) {
	if f.listLinksFn == nil {
		return nil, nil
	}
	return f.listLinksFn(ctx, deviceID)
}

func (f fakeDeviceQueries) ListDeviceChangeEvents(ctx context.Context, arg sqlcgen.ListDeviceChangeEventsParams) ([]sqlcgen.DeviceChangeEvent, error) {
	if f.listChangeEventsFn == nil {
		return nil, nil
	}
	return f.listChangeEventsFn(ctx, arg)
}

func (f fakeDeviceQueries) ListDeviceChangeEventsForDevice(ctx context.Context, arg sqlcgen.ListDeviceChangeEventsForDeviceParams) ([]sqlcgen.DeviceChangeEvent, error) {
	if f.listHistoryFn == nil {
		return nil, nil
	}
	return f.listHistoryFn(ctx, arg)
}

type fakeDiscoveryQueries struct {
	insertFn    func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	updateFn    func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	getLatestFn func(ctx context.Context) (sqlcgen.DiscoveryRun, error)
	getFn       func(ctx context.Context, id string) (sqlcgen.DiscoveryRun, error)
	insertLogFn func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error
	listRunsFn  func(ctx context.Context, arg sqlcgen.ListDiscoveryRunsParams) ([]sqlcgen.DiscoveryRun, error)
	listLogFn   func(ctx context.Context, arg sqlcgen.ListDiscoveryRunLogsParams) ([]sqlcgen.DiscoveryRunLog, error)
}

func (f fakeDiscoveryQueries) InsertDiscoveryRun(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
	if f.insertFn == nil {
		return sqlcgen.DiscoveryRun{}, nil
	}
	return f.insertFn(ctx, arg)
}

func (f fakeDiscoveryQueries) UpdateDiscoveryRun(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
	if f.updateFn == nil {
		return sqlcgen.DiscoveryRun{}, nil
	}
	return f.updateFn(ctx, arg)
}

func (f fakeDiscoveryQueries) GetLatestDiscoveryRun(ctx context.Context) (sqlcgen.DiscoveryRun, error) {
	if f.getLatestFn == nil {
		return sqlcgen.DiscoveryRun{}, nil
	}
	return f.getLatestFn(ctx)
}

func (f fakeDiscoveryQueries) GetDiscoveryRun(ctx context.Context, id string) (sqlcgen.DiscoveryRun, error) {
	if f.getFn == nil {
		return sqlcgen.DiscoveryRun{}, nil
	}
	return f.getFn(ctx, id)
}

func (f fakeDiscoveryQueries) InsertDiscoveryRunLog(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error {
	if f.insertLogFn == nil {
		return nil
	}
	return f.insertLogFn(ctx, arg)
}

func (f fakeDiscoveryQueries) ListDiscoveryRuns(ctx context.Context, arg sqlcgen.ListDiscoveryRunsParams) ([]sqlcgen.DiscoveryRun, error) {
	if f.listRunsFn == nil {
		return nil, nil
	}
	return f.listRunsFn(ctx, arg)
}

func (f fakeDiscoveryQueries) ListDiscoveryRunLogs(ctx context.Context, arg sqlcgen.ListDiscoveryRunLogsParams) ([]sqlcgen.DiscoveryRunLog, error) {
	if f.listLogFn == nil {
		return nil, nil
	}
	return f.listLogFn(ctx, arg)
}

type fakeAuditQueries struct {
	insertFn func(ctx context.Context, arg sqlcgen.InsertAuditEventParams) error
}

func (f fakeAuditQueries) InsertAuditEvent(ctx context.Context, arg sqlcgen.InsertAuditEventParams) error {
	if f.insertFn == nil {
		return nil
	}
	return f.insertFn(ctx, arg)
}

func decodeBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatalf("failed to decode body as json: %v\nbody=%s", err, rr.Body.String())
	}
	return v
}

func TestDevices_List_OK(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		listFn: func(ctx context.Context) ([]sqlcgen.Device, error) {
			name := "router"
			return []sqlcgen.Device{{ID: "00000000-0000-0000-0000-000000000001", DisplayName: &name}}, nil
		},
		getFn:    func(ctx context.Context, id string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		createFn: func(ctx context.Context, displayName *string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)

	h.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("expected json content-type, got %q", got)
	}

	// Request ID should be set in responses by middleware.
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected X-Request-ID header to be set")
	}
}

func TestDevices_Export_Attachment(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	name := "gateway"
	h.devices = fakeDeviceQueries{
		listFn: func(ctx context.Context) ([]sqlcgen.Device, error) {
			return []sqlcgen.Device{{ID: "00000000-0000-0000-0000-000000000020", DisplayName: &name}}, nil
		},
		getFn:    func(ctx context.Context, id string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		createFn: func(ctx context.Context, displayName *string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/export", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if cd := rr.Header().Get("Content-Disposition"); cd != "attachment; filename=\"roller_hoops_devices.json\"" {
		t.Fatalf("expected Content-Disposition attachment, got %q", cd)
	}
}

func TestDevices_Get_NotFound(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		listFn: func(ctx context.Context) ([]sqlcgen.Device, error) { return nil, nil },
		getFn: func(ctx context.Context, id string) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, pgx.ErrNoRows
		},
		createFn: func(ctx context.Context, displayName *string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/00000000-0000-0000-0000-000000000002", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error envelope, got: %v", body)
	}
	if errObj["code"] != "not_found" {
		t.Fatalf("expected not_found, got %v", errObj["code"])
	}
}

func TestDevices_Get_InvalidID(t *testing.T) {
	invalidUUIDErr := &pgconn.PgError{Code: "22P02", Message: "invalid input syntax for type uuid"}

	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		listFn: func(ctx context.Context) ([]sqlcgen.Device, error) { return nil, nil },
		getFn: func(ctx context.Context, id string) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, invalidUUIDErr
		},
		createFn: func(ctx context.Context, displayName *string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/not-a-uuid", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "invalid_id" {
		t.Fatalf("expected invalid_id, got %v", errObj["code"])
	}
}

func TestDevices_Create_RejectsUnknownFields(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		listFn: func(ctx context.Context) ([]sqlcgen.Device, error) { return nil, nil },
		getFn:  func(ctx context.Context, id string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		createFn: func(ctx context.Context, displayName *string) (sqlcgen.Device, error) {
			return sqlcgen.Device{ID: "00000000-0000-0000-0000-000000000003", DisplayName: displayName}, nil
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(`{"display_name":"x","nope":true}`))
	req.Header.Set("Content-Type", "application/json")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "validation_failed" {
		t.Fatalf("expected validation_failed, got %v", errObj["code"])
	}
}

func TestDevices_Create_WithMetadata(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		listFn: func(ctx context.Context) ([]sqlcgen.Device, error) { return nil, nil },
		getFn:  func(ctx context.Context, id string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		createFn: func(ctx context.Context, displayName *string) (sqlcgen.Device, error) {
			return sqlcgen.Device{ID: "00000000-0000-0000-0000-000000000005", DisplayName: displayName}, nil
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, nil
		},
		upsertFn: func(ctx context.Context, arg sqlcgen.UpsertDeviceMetadataParams) (sqlcgen.DeviceMetadata, error) {
			owner := ""
			if arg.Owner != nil {
				owner = *arg.Owner
			}
			if owner != "alice" {
				t.Fatalf("expected owner to be trimmed to alice, got %q", owner)
			}
			return sqlcgen.DeviceMetadata{
				DeviceID: arg.DeviceID,
				Owner:    arg.Owner,
				Location: arg.Location,
				Notes:    arg.Notes,
			}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(`{"display_name":"router","metadata":{"owner":" alice ","notes":" diag "}}`))
	req.Header.Set("Content-Type", "application/json")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	meta, ok := body["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata in response, got: %v", body)
	}
	if meta["owner"] != "alice" {
		t.Fatalf("expected owner to be alice, got %v", meta["owner"])
	}
	if meta["notes"] != "diag" {
		t.Fatalf("expected notes to be diag, got %v", meta["notes"])
	}
}

func TestDevices_Create_UsesUpstreamRequestID(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		listFn: func(ctx context.Context) ([]sqlcgen.Device, error) { return nil, nil },
		getFn:  func(ctx context.Context, id string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		createFn: func(ctx context.Context, displayName *string) (sqlcgen.Device, error) {
			return sqlcgen.Device{ID: "00000000-0000-0000-0000-000000000004", DisplayName: displayName}, nil
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(`{"display_name":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	// Intentionally use the canonical header name configured by chi.
	req.Header.Set("X-Request-ID", "req-123")

	h.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	if got := rr.Header().Get("X-Request-ID"); got != "req-123" {
		t.Fatalf("expected request id to be preserved, got %q", got)
	}
}

func TestDevices_Import_CreatesAndUpdates(t *testing.T) {
	created := 0
	updated := 0
	metadataCalls := make([]sqlcgen.UpsertDeviceMetadataParams, 0, 2)
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		listFn: func(ctx context.Context) ([]sqlcgen.Device, error) { return nil, nil },
		getFn:  func(ctx context.Context, id string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		createFn: func(ctx context.Context, displayName *string) (sqlcgen.Device, error) {
			created++
			return sqlcgen.Device{ID: "imported-1", DisplayName: displayName}, nil
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error) {
			updated++
			if arg.ID != "00000000-0000-0000-0000-000000000030" {
				t.Fatalf("unexpected device id: %s", arg.ID)
			}
			return sqlcgen.Device{ID: arg.ID, DisplayName: arg.DisplayName}, nil
		},
		upsertFn: func(ctx context.Context, arg sqlcgen.UpsertDeviceMetadataParams) (sqlcgen.DeviceMetadata, error) {
			metadataCalls = append(metadataCalls, arg)
			return sqlcgen.DeviceMetadata{
				DeviceID: arg.DeviceID,
				Owner:    arg.Owner,
				Location: arg.Location,
				Notes:    arg.Notes,
			}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/devices/import", strings.NewReader(`{
	  "devices": [
	    {"display_name": "imported device", "metadata": {"owner": " bob ", "notes": " info "}},
	    {"id": "00000000-0000-0000-0000-000000000030", "display_name": "existing", "metadata": {"location": " rack "}}
	  ]
	}`))
	req.Header.Set("Content-Type", "application/json")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode import response: %v", err)
	}

	if body["created"] != float64(1) {
		t.Fatalf("expected created=1, got %v", body["created"])
	}
	if body["updated"] != float64(1) {
		t.Fatalf("expected updated=1, got %v", body["updated"])
	}
	if len(metadataCalls) != 2 {
		t.Fatalf("expected 2 metadata upserts, got %d", len(metadataCalls))
	}
	if owner := metadataCalls[0].Owner; owner == nil || *owner != "bob" {
		t.Fatalf("expected owner trimmed to bob, got %v", owner)
	}
	if location := metadataCalls[1].Location; location == nil || *location != "rack" {
		t.Fatalf("expected location trimmed to rack, got %v", location)
	}
}

func TestDiscovery_Status_Idle(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.discovery = fakeDiscoveryQueries{
		getLatestFn: func(ctx context.Context) (sqlcgen.DiscoveryRun, error) {
			return sqlcgen.DiscoveryRun{}, pgx.ErrNoRows
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/status", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	if body["status"] != "idle" {
		t.Fatalf("expected idle status, got %v", body)
	}
}

func TestDiscovery_Run_StartsRun(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	now := time.Now()
	h.discovery = fakeDiscoveryQueries{
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			if arg.Status != "queued" {
				t.Fatalf("expected queued status on insert, got %q", arg.Status)
			}
			if arg.Stats == nil || arg.Stats["stage"] != "queued" || arg.Stats["preset"] != "fast" {
				t.Fatalf("expected queued stage + fast preset, got %#v", arg.Stats)
			}
			tags, ok := arg.Stats["tags"].([]string)
			if !ok || len(tags) != 2 || tags[0] != "ports" || tags[1] != "snmp" {
				t.Fatalf("expected tags [ports snmp], got %#v", arg.Stats["tags"])
			}
			return sqlcgen.DiscoveryRun{
				ID:        "run-1",
				Status:    arg.Status,
				Scope:     arg.Scope,
				Stats:     arg.Stats,
				StartedAt: now,
			}, nil
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			return sqlcgen.DiscoveryRun{
				ID:          arg.ID,
				Status:      arg.Status,
				Scope:       nil,
				Stats:       arg.Stats,
				StartedAt:   now,
				CompletedAt: arg.CompletedAt,
			}, nil
		},
		insertLogFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error { return nil },
		getLatestFn: func(ctx context.Context) (sqlcgen.DiscoveryRun, error) { return sqlcgen.DiscoveryRun{}, nil },
		getFn:       func(ctx context.Context, id string) (sqlcgen.DiscoveryRun, error) { return sqlcgen.DiscoveryRun{}, nil },
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/run", strings.NewReader(`{"scope":"10.0.0.0/24","preset":"fast","tags":["snmp","ports"]}`))
	req.Header.Set("Content-Type", "application/json")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	if body["status"] != "queued" {
		t.Fatalf("expected queued status, got %v", body["status"])
	}
	if body["scope"] != "10.0.0.0/24" {
		t.Fatalf("expected scope to round-trip, got %v", body["scope"])
	}
	stats, ok := body["stats"].(map[string]any)
	if !ok || stats["preset"] != "fast" {
		t.Fatalf("expected preset to round-trip via stats, got %v", body["stats"])
	}
	tags, ok := stats["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "ports" || tags[1] != "snmp" {
		t.Fatalf("expected tags to round-trip via stats, got %v", stats["tags"])
	}
	if _, ok := body["id"]; !ok {
		t.Fatalf("expected a run id, got %v", body)
	}
}

func TestDiscovery_Run_RejectsInvalidTags(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.discovery = fakeDiscoveryQueries{
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			t.Fatalf("expected request validation to fail before insert")
			return sqlcgen.DiscoveryRun{}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/run", strings.NewReader(`{"preset":"normal","tags":["banana"]}`))
	req.Header.Set("Content-Type", "application/json")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "validation_failed" {
		t.Fatalf("expected validation_failed, got %v", errObj["code"])
	}
}

func TestDiscovery_ScopeSuggestions_OK(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/scope-suggestions", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	if _, ok := body["scopes"]; !ok {
		t.Fatalf("expected scopes field, got %v", body)
	}
}

func TestDiscovery_Run_RejectsUnknownFields(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.discovery = fakeDiscoveryQueries{
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			// Should not be reached.
			t.Fatalf("expected request validation to fail before insert")
			return sqlcgen.DiscoveryRun{}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/run", strings.NewReader(`{"scope":"10.0.0.0/24","nope":true}`))
	req.Header.Set("Content-Type", "application/json")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "validation_failed" {
		t.Fatalf("expected validation_failed, got %v", errObj["code"])
	}
}

func TestDiscovery_Run_RejectsInvalidPreset(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.discovery = fakeDiscoveryQueries{
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			t.Fatalf("expected request validation to fail before insert")
			return sqlcgen.DiscoveryRun{}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/run", strings.NewReader(`{"preset":"turbo"}`))
	req.Header.Set("Content-Type", "application/json")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "validation_failed" {
		t.Fatalf("expected validation_failed, got %v", errObj["code"])
	}
}

func TestDiscovery_Run_AllowsEmptyBody(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	now := time.Now()

	h.discovery = fakeDiscoveryQueries{
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			if arg.Scope != nil {
				t.Fatalf("expected nil scope when body omitted, got %v", *arg.Scope)
			}
			if arg.Stats == nil || arg.Stats["preset"] != "normal" {
				t.Fatalf("expected normal preset, got %#v", arg.Stats)
			}
			return sqlcgen.DiscoveryRun{ID: "run-empty", Status: arg.Status, Scope: arg.Scope, Stats: arg.Stats, StartedAt: now}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/run", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	if body["status"] != "queued" {
		t.Fatalf("expected queued status, got %v", body["status"])
	}
}

func TestDiscovery_Run_DefaultsScopeWhenConfigured(t *testing.T) {
	defaultScope := "10.0.0.0/24"
	h := NewHandlerWithOptions(NewLogger("debug"), nil, nil, Options{DiscoveryDefaultScope: &defaultScope})
	now := time.Now()

	h.discovery = fakeDiscoveryQueries{
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			if arg.Scope == nil || *arg.Scope != defaultScope {
				t.Fatalf("expected default scope %q, got %v", defaultScope, arg.Scope)
			}
			return sqlcgen.DiscoveryRun{ID: "run-default-scope", Status: arg.Status, Scope: arg.Scope, Stats: arg.Stats, StartedAt: now}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/run", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	if body["scope"] != defaultScope {
		t.Fatalf("expected scope %q, got %v", defaultScope, body["scope"])
	}
}

func TestDiscovery_Run_TrimsScope(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	now := time.Now()

	h.discovery = fakeDiscoveryQueries{
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			if arg.Scope == nil || *arg.Scope != "10.0.0.0/24" {
				t.Fatalf("expected trimmed scope 10.0.0.0/24, got %v", arg.Scope)
			}
			if arg.Stats == nil || arg.Stats["preset"] != "normal" {
				t.Fatalf("expected normal preset, got %#v", arg.Stats)
			}
			return sqlcgen.DiscoveryRun{ID: "run-trim", Status: arg.Status, Scope: arg.Scope, Stats: arg.Stats, StartedAt: now}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/run", strings.NewReader(`{"scope":" 10.0.0.0/24  "}`))
	req.Header.Set("Content-Type", "application/json")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestDiscovery_Status_ReturnsLatestRun(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	now := time.Now()
	completed := now.Add(1 * time.Second)
	lastErr := ""

	h.discovery = fakeDiscoveryQueries{
		getLatestFn: func(ctx context.Context) (sqlcgen.DiscoveryRun, error) {
			scope := "10.0.0.0/24"
			return sqlcgen.DiscoveryRun{
				ID:          "run-99",
				Status:      "succeeded",
				Scope:       &scope,
				Stats:       map[string]any{"devices_seen": 1},
				StartedAt:   now,
				CompletedAt: &completed,
				LastError:   &lastErr,
			}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/status", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	if body["status"] != "succeeded" {
		t.Fatalf("expected succeeded status, got %v", body["status"])
	}
	latest, ok := body["latest_run"].(map[string]any)
	if !ok {
		t.Fatalf("expected latest_run object, got %v", body)
	}
	if latest["id"] != "run-99" {
		t.Fatalf("expected latest_run.id run-99, got %v", latest["id"])
	}
}

func TestDevices_Changes_ReturnsCursor(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	now := time.Now().UTC()
	h.devices = fakeDeviceQueries{
		listChangeEventsFn: func(ctx context.Context, arg sqlcgen.ListDeviceChangeEventsParams) ([]sqlcgen.DeviceChangeEvent, error) {
			return []sqlcgen.DeviceChangeEvent{
				{EventID: "evt-2", DeviceID: "dev-1", EventAt: now, Kind: "ip_observation", Summary: "10.0.0.1", Details: map[string]any{"ip": "10.0.0.1"}},
				{EventID: "evt-1", DeviceID: "dev-1", EventAt: now.Add(-time.Minute), Kind: "mac_observation", Summary: "aa:bb", Details: map[string]any{"mac": "aa:bb"}},
			}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/changes?limit=1", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp deviceChangeEventsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(resp.Events))
	}
	if resp.Cursor == nil || *resp.Cursor == "" {
		t.Fatalf("expected cursor to be present")
	}
}

func TestDevices_History_NotFound(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		getFn: func(ctx context.Context, id string) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, pgx.ErrNoRows
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/00000000-0000-0000-0000-000000000100/history", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}

	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "not_found" {
		t.Fatalf("expected not_found error code, got %v", errObj["code"])
	}
}

func TestDevices_History_Paginates(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	now := time.Now().UTC()
	h.devices = fakeDeviceQueries{
		getFn: func(ctx context.Context, id string) (sqlcgen.Device, error) {
			return sqlcgen.Device{ID: id}, nil
		},
		listHistoryFn: func(ctx context.Context, arg sqlcgen.ListDeviceChangeEventsForDeviceParams) ([]sqlcgen.DeviceChangeEvent, error) {
			return []sqlcgen.DeviceChangeEvent{
				{EventID: "evt-a", DeviceID: arg.DeviceID, EventAt: now, Kind: "metadata", Summary: "updated", Details: map[string]any{"owner": "alice"}},
				{EventID: "evt-b", DeviceID: arg.DeviceID, EventAt: now.Add(-time.Minute), Kind: "ip", Summary: "10.0.0.2"},
			}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices/00000000-0000-0000-0000-000000000100/history?limit=1", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp deviceChangeEventsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(resp.Events))
	}
	if resp.Cursor == nil || *resp.Cursor == "" {
		t.Fatalf("expected cursor to be present")
	}
}

func TestDiscovery_Runs_Pagination(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	now := time.Now().UTC()
	h.discovery = fakeDiscoveryQueries{
		listRunsFn: func(ctx context.Context, arg sqlcgen.ListDiscoveryRunsParams) ([]sqlcgen.DiscoveryRun, error) {
			return []sqlcgen.DiscoveryRun{
				{ID: "run-1", Status: "succeeded", StartedAt: now, Stats: map[string]any{}},
				{ID: "run-0", Status: "failed", StartedAt: now.Add(-time.Minute), Stats: map[string]any{}},
			}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/runs?limit=1", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp discoveryRunPage
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(resp.Runs))
	}
	if resp.Cursor == nil || *resp.Cursor == "" {
		t.Fatalf("expected cursor")
	}
}

func TestDiscovery_RunLogs_Pagination(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	now := time.Now().UTC()
	h.discovery = fakeDiscoveryQueries{
		getFn: func(ctx context.Context, id string) (sqlcgen.DiscoveryRun, error) {
			return sqlcgen.DiscoveryRun{ID: id}, nil
		},
		listLogFn: func(ctx context.Context, arg sqlcgen.ListDiscoveryRunLogsParams) ([]sqlcgen.DiscoveryRunLog, error) {
			return []sqlcgen.DiscoveryRunLog{
				{ID: 2, RunID: arg.RunID, Level: "info", Message: "first", CreatedAt: now},
				{ID: 1, RunID: arg.RunID, Level: "error", Message: "second", CreatedAt: now.Add(-time.Minute)},
			}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/runs/run-42/logs?limit=1", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var resp discoveryRunLogPage
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(resp.Logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(resp.Logs))
	}
	if resp.Cursor == nil {
		t.Fatalf("expected cursor")
	}
}

func TestAudit_Create_OK(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	seen := false
	h.audit = fakeAuditQueries{
		insertFn: func(ctx context.Context, arg sqlcgen.InsertAuditEventParams) error {
			seen = true
			if arg.Actor != "alice" || arg.Action != "device.create" {
				t.Fatalf("unexpected audit payload: %#v", arg)
			}
			if arg.Details == nil || arg.Details["x"] != float64(1) {
				t.Fatalf("expected details to include x=1, got %#v", arg.Details)
			}
			return nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/audit/events", strings.NewReader(`{"actor":"alice","action":"device.create","details":{"x":1}}`))
	req.Header.Set("Content-Type", "application/json")

	h.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if !seen {
		t.Fatalf("expected InsertAuditEvent to be called")
	}
}

func TestMetrics_Endpoint_Exposed_WhenEnabled(t *testing.T) {
	m := metrics.New()
	// Prometheus will often omit metric families that have no samples yet.
	// Touch the counter once so the scrape output is deterministic.
	m.ObserveHTTPRequest(http.MethodGet, "/healthz", http.StatusOK, 10*time.Millisecond)

	h := NewHandlerWithMetrics(NewLogger("debug"), nil, m)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	h.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got == "" {
		t.Fatalf("expected content-type to be set")
	}
	if !strings.Contains(rr.Body.String(), "roller_http_requests_total") {
		t.Fatalf("expected roller_http_requests_total in body")
	}
}

func TestMetrics_Endpoint_NotExposed_WhenDisabled(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	h.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}
