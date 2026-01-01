package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"roller_hoops/core-go/internal/sqlcgen"
)

func TestMapProjection_NoFocus_ReturnsEmptyProjection(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	if body["layer"] != "l3" {
		t.Fatalf("expected layer l3, got %v", body["layer"])
	}
	if guidance, ok := body["guidance"].(string); !ok || guidance == "" {
		t.Fatalf("expected guidance string to be present, got %T %v", body["guidance"], body["guidance"])
	}
	if regions, ok := body["regions"].([]any); !ok || len(regions) != 0 {
		t.Fatalf("expected empty regions array, got %T len=%d", body["regions"], len(regions))
	}
	if nodes, ok := body["nodes"].([]any); !ok || len(nodes) != 0 {
		t.Fatalf("expected empty nodes array, got %T len=%d", body["nodes"], len(nodes))
	}
	if edges, ok := body["edges"].([]any); !ok || len(edges) != 0 {
		t.Fatalf("expected empty edges array, got %T len=%d", body["edges"], len(edges))
	}
	trunc, ok := body["truncation"].(map[string]any)
	if !ok || trunc == nil {
		t.Fatalf("expected truncation to be present, got %T %v", body["truncation"], body["truncation"])
	}
	for _, key := range []string{"regions", "nodes", "edges"} {
		metric, ok := trunc[key].(map[string]any)
		if !ok || metric == nil {
			t.Fatalf("expected truncation.%s metric object, got %T %v", key, trunc[key], trunc[key])
		}
		if _, ok := metric["returned"].(float64); !ok {
			t.Fatalf("expected truncation.%s.returned number, got %T %v", key, metric["returned"], metric["returned"])
		}
		if _, ok := metric["limit"].(float64); !ok {
			t.Fatalf("expected truncation.%s.limit number, got %T %v", key, metric["limit"], metric["limit"])
		}
		if _, ok := metric["truncated"].(bool); !ok {
			t.Fatalf("expected truncation.%s.truncated bool, got %T %v", key, metric["truncated"], metric["truncated"])
		}
	}
}

func TestMapProjection_InvalidLayer_Returns400(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/banana", nil)
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

func TestMapProjection_InvalidDepth_Returns400(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?depth=-1", nil)
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

func TestMapProjection_InvalidLimit_Returns400(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?limit=0", nil)
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

func TestMapProjection_LimitParam_LowersTruncationLimits(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?limit=7", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	trunc := body["truncation"].(map[string]any)
	nodesMetric := trunc["nodes"].(map[string]any)
	if nodesMetric["limit"] != float64(7) {
		t.Fatalf("expected truncation.nodes.limit=7, got %v", nodesMetric["limit"])
	}
}

func TestMapProjection_FocusTypeWithoutID_Returns400(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=device", nil)
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

func TestMapProjection_DeviceFocus_DBUnavailable_Returns503(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=device&focusId=00000000-0000-0000-0000-000000000010", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "db_unavailable" {
		t.Fatalf("expected db_unavailable, got %v", errObj["code"])
	}
}

func TestMapProjection_SubnetFocus_InvalidCIDR_Returns400(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=subnet&focusId=not-a-cidr", nil)
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

func TestMapProjection_SubnetFocus_DBUnavailable_Returns503(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=subnet&focusId=10.0.1.0/24", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "db_unavailable" {
		t.Fatalf("expected db_unavailable, got %v", errObj["code"])
	}
}

func TestMapProjection_DeviceFocus_InvalidID_Returns400(t *testing.T) {
	invalidUUIDErr := &pgconn.PgError{Code: "22P02", Message: "invalid input syntax for type uuid"}

	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		getFn: func(ctx context.Context, id string) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, invalidUUIDErr
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=device&focusId=not-a-uuid", nil)
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

func TestMapProjection_DeviceFocus_NotFound_Returns404(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		getFn: func(ctx context.Context, id string) (sqlcgen.Device, error) {
			return sqlcgen.Device{}, pgx.ErrNoRows
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=device&focusId=00000000-0000-0000-0000-000000000010", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "not_found" {
		t.Fatalf("expected not_found, got %v", errObj["code"])
	}
}

func TestMapProjection_DeviceFocus_EchoesFocusAndInspector(t *testing.T) {
	name := "router-1"
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		getFn: func(ctx context.Context, id string) (sqlcgen.Device, error) {
			return sqlcgen.Device{ID: "00000000-0000-0000-0000-000000000011", DisplayName: &name}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=device&focusId=00000000-0000-0000-0000-000000000011", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	focus := body["focus"].(map[string]any)
	if focus["type"] != "device" {
		t.Fatalf("expected focus.type=device, got %v", focus["type"])
	}
	if focus["id"] != "00000000-0000-0000-0000-000000000011" {
		t.Fatalf("expected focus.id to match, got %v", focus["id"])
	}
	if body["inspector"] == nil {
		t.Fatalf("expected inspector to be present")
	}
}

func TestMapProjection_DeviceFocus_L3InspectorIncludesSubnetRelationships(t *testing.T) {
	name := "router-1"
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueries{
		getFn: func(ctx context.Context, id string) (sqlcgen.Device, error) {
			return sqlcgen.Device{ID: "00000000-0000-0000-0000-000000000011", DisplayName: &name}, nil
		},
		listIPsFn: func(ctx context.Context, deviceID string) ([]sqlcgen.DeviceIP, error) {
			return []sqlcgen.DeviceIP{
				{IP: "10.0.2.20"},
				{IP: "10.0.1.10"},
			}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=device&focusId=00000000-0000-0000-0000-000000000011", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	inspector, ok := body["inspector"].(map[string]any)
	if !ok || inspector == nil {
		t.Fatalf("expected inspector object, got %T %v", body["inspector"], body["inspector"])
	}
	relsAny, ok := inspector["relationships"].([]any)
	if !ok {
		t.Fatalf("expected inspector.relationships array, got %T %v", inspector["relationships"], inspector["relationships"])
	}

	want := map[string]bool{
		"10.0.1.0/24": false,
		"10.0.2.0/24": false,
	}
	for _, relAny := range relsAny {
		rel, ok := relAny.(map[string]any)
		if !ok {
			continue
		}
		if rel["layer"] != "l3" || rel["focus_type"] != "subnet" {
			continue
		}
		if id, ok := rel["focus_id"].(string); ok {
			if _, exists := want[id]; exists {
				want[id] = true
			}
		}
	}
	for subnet, found := range want {
		if !found {
			t.Fatalf("expected inspector relationships to include subnet %s", subnet)
		}
	}
}

type fakeDeviceQueriesWithCIDR struct {
	fakeDeviceQueries
	listCIDRFn func(ctx context.Context, cidr string, limit int32) ([]sqlcgen.MapDevicePeer, error)
}

func (f fakeDeviceQueriesWithCIDR) ListDevicesInCIDR(ctx context.Context, cidr string, limit int32) ([]sqlcgen.MapDevicePeer, error) {
	if f.listCIDRFn == nil {
		return nil, nil
	}
	return f.listCIDRFn(ctx, cidr, limit)
}

func TestMapProjection_SubnetFocus_L3InspectorIncludesDeviceRelationships(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.devices = fakeDeviceQueriesWithCIDR{
		fakeDeviceQueries: fakeDeviceQueries{
			getFn: func(ctx context.Context, id string) (sqlcgen.Device, error) { return sqlcgen.Device{}, nil },
		},
		listCIDRFn: func(ctx context.Context, cidr string, limit int32) ([]sqlcgen.MapDevicePeer, error) {
			name1 := "peer-1"
			name2 := "peer-2"
			return []sqlcgen.MapDevicePeer{
				{ID: "00000000-0000-0000-0000-000000000001", DisplayName: &name1},
				{ID: "00000000-0000-0000-0000-000000000002", DisplayName: &name2},
			}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=subnet&focusId=10.0.1.5/24", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := decodeBody(t, rr)
	inspector, ok := body["inspector"].(map[string]any)
	if !ok || inspector == nil {
		t.Fatalf("expected inspector object, got %T %v", body["inspector"], body["inspector"])
	}
	relsAny, ok := inspector["relationships"].([]any)
	if !ok {
		t.Fatalf("expected inspector.relationships array, got %T %v", inspector["relationships"], inspector["relationships"])
	}

	want := map[string]bool{
		"00000000-0000-0000-0000-000000000001": false,
		"00000000-0000-0000-0000-000000000002": false,
	}
	for _, relAny := range relsAny {
		rel, ok := relAny.(map[string]any)
		if !ok {
			continue
		}
		if rel["layer"] != "l3" || rel["focus_type"] != "device" {
			continue
		}
		if id, ok := rel["focus_id"].(string); ok {
			if _, exists := want[id]; exists {
				want[id] = true
			}
		}
	}
	for deviceID, found := range want {
		if !found {
			t.Fatalf("expected inspector relationships to include device %s", deviceID)
		}
	}
}
