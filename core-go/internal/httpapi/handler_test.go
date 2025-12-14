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

	"roller_hoops/core-go/internal/sqlcgen"
)

type fakeDeviceQueries struct {
	listFn               func(ctx context.Context) ([]sqlcgen.Device, error)
	getFn                func(ctx context.Context, id string) (sqlcgen.Device, error)
	createFn             func(ctx context.Context, displayName *string) (sqlcgen.Device, error)
	updateFn             func(ctx context.Context, arg sqlcgen.UpdateDeviceParams) (sqlcgen.Device, error)
	upsertFn             func(ctx context.Context, arg sqlcgen.UpsertDeviceMetadataParams) (sqlcgen.DeviceMetadata, error)
	listNameCandidatesFn func(ctx context.Context, deviceID string) ([]sqlcgen.DeviceNameCandidate, error)
}

func (f fakeDeviceQueries) ListDevices(ctx context.Context) ([]sqlcgen.Device, error) {
	return f.listFn(ctx)
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

func (f fakeDeviceQueries) ListDeviceNameCandidates(ctx context.Context, deviceID string) ([]sqlcgen.DeviceNameCandidate, error) {
	if f.listNameCandidatesFn == nil {
		return nil, nil
	}
	return f.listNameCandidatesFn(ctx, deviceID)
}

type fakeDiscoveryQueries struct {
	insertFn    func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	updateFn    func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	getLatestFn func(ctx context.Context) (sqlcgen.DiscoveryRun, error)
	getFn       func(ctx context.Context, id string) (sqlcgen.DiscoveryRun, error)
	insertLogFn func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error
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
	req := httptest.NewRequest(http.MethodPost, "/api/v1/discovery/run", strings.NewReader(`{"scope":"10.0.0.0/24"}`))
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
	if _, ok := body["id"]; !ok {
		t.Fatalf("expected a run id, got %v", body)
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

func TestDiscovery_Run_AllowsEmptyBody(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	now := time.Now()

	h.discovery = fakeDiscoveryQueries{
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			if arg.Scope != nil {
				t.Fatalf("expected nil scope when body omitted, got %v", *arg.Scope)
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

func TestDiscovery_Run_TrimsScope(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	now := time.Now()

	h.discovery = fakeDiscoveryQueries{
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			if arg.Scope == nil || *arg.Scope != "10.0.0.0/24" {
				t.Fatalf("expected trimmed scope 10.0.0.0/24, got %v", arg.Scope)
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
