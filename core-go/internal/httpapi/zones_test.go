package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"roller_hoops/core-go/internal/sqlcgen"
)

type fakeTopologyQueries struct {
	listZonesFn          func(ctx context.Context) ([]sqlcgen.Zone, error)
	getZoneFn            func(ctx context.Context, id string) (sqlcgen.Zone, error)
	createZoneFn         func(ctx context.Context, arg sqlcgen.CreateZoneParams) (sqlcgen.Zone, error)
	updateZoneFn         func(ctx context.Context, arg sqlcgen.UpdateZoneParams) (sqlcgen.Zone, error)
	deleteZoneFn         func(ctx context.Context, id string) (int64, error)
	listZoneMembersFn    func(ctx context.Context, zoneID string) ([]sqlcgen.ZoneMember, error)
	addZoneMemberFn      func(ctx context.Context, arg sqlcgen.AddZoneMemberParams) (int64, error)
	removeZoneMemberFn   func(ctx context.Context, arg sqlcgen.RemoveZoneMemberParams) (int64, error)
	replaceMembersFn     func(ctx context.Context, arg sqlcgen.ReplaceZoneMembersParams) error
	listExistingIDsFn    func(ctx context.Context, deviceIDs []string) ([]string, error)
}

func (f fakeTopologyQueries) ListZones(ctx context.Context) ([]sqlcgen.Zone, error) {
	if f.listZonesFn == nil {
		return nil, nil
	}
	return f.listZonesFn(ctx)
}

func (f fakeTopologyQueries) GetZone(ctx context.Context, id string) (sqlcgen.Zone, error) {
	if f.getZoneFn == nil {
		return sqlcgen.Zone{}, pgx.ErrNoRows
	}
	return f.getZoneFn(ctx, id)
}

func (f fakeTopologyQueries) CreateZone(ctx context.Context, arg sqlcgen.CreateZoneParams) (sqlcgen.Zone, error) {
	if f.createZoneFn == nil {
		return sqlcgen.Zone{}, nil
	}
	return f.createZoneFn(ctx, arg)
}

func (f fakeTopologyQueries) UpdateZone(ctx context.Context, arg sqlcgen.UpdateZoneParams) (sqlcgen.Zone, error) {
	if f.updateZoneFn == nil {
		return sqlcgen.Zone{}, nil
	}
	return f.updateZoneFn(ctx, arg)
}

func (f fakeTopologyQueries) DeleteZone(ctx context.Context, id string) (int64, error) {
	if f.deleteZoneFn == nil {
		return 0, nil
	}
	return f.deleteZoneFn(ctx, id)
}

func (f fakeTopologyQueries) ListZoneMembers(ctx context.Context, zoneID string) ([]sqlcgen.ZoneMember, error) {
	if f.listZoneMembersFn == nil {
		return nil, nil
	}
	return f.listZoneMembersFn(ctx, zoneID)
}

func (f fakeTopologyQueries) AddZoneMember(ctx context.Context, arg sqlcgen.AddZoneMemberParams) (int64, error) {
	if f.addZoneMemberFn == nil {
		return 0, nil
	}
	return f.addZoneMemberFn(ctx, arg)
}

func (f fakeTopologyQueries) RemoveZoneMember(ctx context.Context, arg sqlcgen.RemoveZoneMemberParams) (int64, error) {
	if f.removeZoneMemberFn == nil {
		return 0, nil
	}
	return f.removeZoneMemberFn(ctx, arg)
}

func (f fakeTopologyQueries) ReplaceZoneMembers(ctx context.Context, arg sqlcgen.ReplaceZoneMembersParams) error {
	if f.replaceMembersFn == nil {
		return nil
	}
	return f.replaceMembersFn(ctx, arg)
}

func (f fakeTopologyQueries) ListExistingDeviceIDs(ctx context.Context, deviceIDs []string) ([]string, error) {
	if f.listExistingIDsFn == nil {
		return nil, nil
	}
	return f.listExistingIDsFn(ctx, deviceIDs)
}

func TestZones_List_OK(t *testing.T) {
	desc := "demilitarized"
	now := time.Now().UTC()
	h := NewHandler(NewLogger("debug"), nil)
	h.topology = fakeTopologyQueries{
		listZonesFn: func(ctx context.Context) ([]sqlcgen.Zone, error) {
			return []sqlcgen.Zone{{
				ID:          "00000000-0000-0000-0000-000000000101",
				Name:        "DMZ",
				Description: &desc,
				CreatedAt:   now,
				UpdatedAt:   now,
			}}, nil
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topology/zones", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	body := rr.Body.String()
	if !strings.Contains(body, "DMZ") {
		t.Fatalf("expected DMZ in body, got %s", body)
	}
}

func TestZones_Create_ForbiddenWithoutAdmin(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.topology = fakeTopologyQueries{}
	h.audit = fakeAuditQueries{}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/topology/zones", strings.NewReader(`{"name":"DMZ"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "read-only")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestZones_Create_OK_Audited(t *testing.T) {
	now := time.Now().UTC()
	var gotAudit sqlcgen.InsertAuditEventParams
	h := NewHandler(NewLogger("debug"), nil)
	h.topology = fakeTopologyQueries{
		createZoneFn: func(ctx context.Context, arg sqlcgen.CreateZoneParams) (sqlcgen.Zone, error) {
			if arg.Name != "DMZ" {
				t.Fatalf("expected trimmed name DMZ, got %q", arg.Name)
			}
			if arg.Description == nil || *arg.Description != "Internet-facing" {
				t.Fatalf("unexpected description: %#v", arg.Description)
			}
			return sqlcgen.Zone{
				ID:          "00000000-0000-0000-0000-000000000102",
				Name:        arg.Name,
				Description: arg.Description,
				CreatedAt:   now,
				UpdatedAt:   now,
			}, nil
		},
	}
	h.audit = fakeAuditQueries{insertFn: func(ctx context.Context, arg sqlcgen.InsertAuditEventParams) error {
		gotAudit = arg
		return nil
	}}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/topology/zones", strings.NewReader(`{"name":"  DMZ ","description":" Internet-facing "}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User", "alice")
	req.Header.Set("X-User-Role", "admin")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if gotAudit.Action != "create_zone" {
		t.Fatalf("expected create_zone audit action, got %q", gotAudit.Action)
	}
	if gotAudit.Actor != "alice" {
		t.Fatalf("expected actor alice, got %q", gotAudit.Actor)
	}
}

func TestZones_Get_NotFound(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.topology = fakeTopologyQueries{
		getZoneFn: func(ctx context.Context, id string) (sqlcgen.Zone, error) {
			return sqlcgen.Zone{}, pgx.ErrNoRows
		},
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/topology/zones/00000000-0000-0000-0000-000000000103", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestZones_ReplaceMembers_OK(t *testing.T) {
	var gotReplace sqlcgen.ReplaceZoneMembersParams
	var gotAudit sqlcgen.InsertAuditEventParams
	h := NewHandler(NewLogger("debug"), nil)
	h.topology = fakeTopologyQueries{
		getZoneFn: func(ctx context.Context, id string) (sqlcgen.Zone, error) {
			return sqlcgen.Zone{ID: id, Name: "Internal"}, nil
		},
		listZoneMembersFn: func(ctx context.Context, zoneID string) ([]sqlcgen.ZoneMember, error) {
			return []sqlcgen.ZoneMember{{DeviceID: "00000000-0000-0000-0000-000000000201"}}, nil
		},
		listExistingIDsFn: func(ctx context.Context, deviceIDs []string) ([]string, error) {
			return []string{"00000000-0000-0000-0000-000000000201", "00000000-0000-0000-0000-000000000202"}, nil
		},
		replaceMembersFn: func(ctx context.Context, arg sqlcgen.ReplaceZoneMembersParams) error {
			gotReplace = arg
			return nil
		},
	}
	h.audit = fakeAuditQueries{insertFn: func(ctx context.Context, arg sqlcgen.InsertAuditEventParams) error {
		gotAudit = arg
		return nil
	}}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/topology/zones/00000000-0000-0000-0000-000000000104/members", strings.NewReader(`{"device_ids":["00000000-0000-0000-0000-000000000201","00000000-0000-0000-0000-000000000202","00000000-0000-0000-0000-000000000202"]}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "admin")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if len(gotReplace.DeviceIDs) != 2 {
		t.Fatalf("expected deduped device ids, got %#v", gotReplace.DeviceIDs)
	}
	if gotAudit.Action != "set_zone_members" {
		t.Fatalf("expected set_zone_members audit action, got %q", gotAudit.Action)
	}
}

func TestZones_AddMember_DeviceNotFound(t *testing.T) {
	h := NewHandler(NewLogger("debug"), nil)
	h.topology = fakeTopologyQueries{
		getZoneFn: func(ctx context.Context, id string) (sqlcgen.Zone, error) {
			return sqlcgen.Zone{ID: id, Name: "Internal"}, nil
		},
		listExistingIDsFn: func(ctx context.Context, deviceIDs []string) ([]string, error) {
			return nil, nil
		},
	}
	h.audit = fakeAuditQueries{}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/topology/zones/00000000-0000-0000-0000-000000000105/members", strings.NewReader(`{"device_id":"00000000-0000-0000-0000-000000000299"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "admin")
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}
