package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"roller_hoops/core-go/internal/db"
)

func requireTestDatabaseURL(t *testing.T) string {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}
	return dsn
}

func mustDeriveDatabaseURL(t *testing.T, baseURL, dbName string) string {
	t.Helper()

	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		t.Skipf("TEST_DATABASE_URL must be a URL-style DSN (e.g. postgres://...); got %q", baseURL)
	}

	u.Path = "/" + dbName
	return u.String()
}

func newTestDatabaseName() string {
	// Safe identifier (letters/digits/underscores) so we can use it without quoting.
	return fmt.Sprintf("roller_hoops_test_%d", time.Now().UnixNano())
}

func createDatabase(ctx context.Context, adminURL, dbName string) error {
	adminConn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer adminConn.Close(ctx)

	_, err = adminConn.Exec(ctx, "CREATE DATABASE "+dbName)
	return err
}

func dropDatabase(ctx context.Context, adminURL, dbName string) error {
	adminConn, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		return err
	}
	defer adminConn.Close(ctx)

	if _, err := adminConn.Exec(ctx, "DROP DATABASE "+dbName+" WITH (FORCE)"); err == nil {
		return nil
	}
	_, err = adminConn.Exec(ctx, "DROP DATABASE "+dbName)
	return err
}

func migrationsDir(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	return filepath.Join(repoRoot, "core-go", "migrations")
}

func applyMigrations(ctx context.Context, conn *pgx.Conn, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var ups []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			ups = append(ups, name)
		}
	}
	sort.Strings(ups)

	for _, name := range ups {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		if _, err := conn.Exec(ctx, string(b)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	return nil
}

func TestHandler_Postgres_DeviceCRUD(t *testing.T) {
	adminURL := requireTestDatabaseURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := newTestDatabaseName()
	testDBURL := mustDeriveDatabaseURL(t, adminURL, dbName)

	if err := createDatabase(ctx, adminURL, dbName); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		_ = dropDatabase(context.Background(), adminURL, dbName)
	})

	mConn, err := pgx.Connect(ctx, testDBURL)
	if err != nil {
		t.Fatalf("connect for migrations: %v", err)
	}
	if err := applyMigrations(ctx, mConn, migrationsDir(t)); err != nil {
		_ = mConn.Close(ctx)
		t.Fatalf("apply migrations: %v", err)
	}
	if err := mConn.Close(ctx); err != nil {
		t.Fatalf("close migration connection: %v", err)
	}

	pool, err := db.Open(ctx, testDBURL)
	if err != nil {
		t.Fatalf("open db pool: %v", err)
	}
	t.Cleanup(pool.Close)

	h := NewHandler(NewLogger("error"), pool)
	router := h.Router()

	rrReady := httptest.NewRecorder()
	router.ServeHTTP(rrReady, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rrReady.Code != http.StatusOK {
		t.Fatalf("readyz expected 200, got %d: %s", rrReady.Code, rrReady.Body.String())
	}

	rrCreate := httptest.NewRecorder()
	reqCreate := httptest.NewRequest(http.MethodPost, "/api/v1/devices", strings.NewReader(`{"display_name":"integration-router","metadata":{"owner":"alice","location":"lab","notes":"hi"}}`))
	reqCreate.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rrCreate, reqCreate)
	if rrCreate.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d: %s", rrCreate.Code, rrCreate.Body.String())
	}

	var created device
	if err := json.NewDecoder(rrCreate.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected created device id to be set")
	}
	if created.Metadata == nil || created.Metadata.Owner == nil || *created.Metadata.Owner != "alice" {
		t.Fatalf("expected metadata.owner to persist, got %+v", created.Metadata)
	}

	// Seed IP observations with deterministic timestamps so we can validate primary_ip selection.
	seedConn, err := pgx.Connect(ctx, testDBURL)
	if err != nil {
		t.Fatalf("connect for seed data: %v", err)
	}
	seedOld := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	seedNew := seedOld.Add(1 * time.Hour)
	if _, err := seedConn.Exec(
		ctx,
		`INSERT INTO ip_addresses (device_id, ip, created_at, updated_at) VALUES ($1::uuid, $2::inet, $3::timestamptz, $3::timestamptz)`,
		created.ID,
		"192.0.2.10",
		seedOld,
	); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert ip_addresses seed (old): %v", err)
	}
	if _, err := seedConn.Exec(
		ctx,
		`INSERT INTO ip_addresses (device_id, ip, created_at, updated_at) VALUES ($1::uuid, $2::inet, $3::timestamptz, $3::timestamptz)`,
		created.ID,
		"192.0.2.20",
		seedNew,
	); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert ip_addresses seed (new): %v", err)
	}
	if err := seedConn.Close(ctx); err != nil {
		t.Fatalf("close seed connection: %v", err)
	}

	rrList := httptest.NewRecorder()
	router.ServeHTTP(rrList, httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil))
	if rrList.Code != http.StatusOK {
		t.Fatalf("list expected 200, got %d: %s", rrList.Code, rrList.Body.String())
	}

	var page devicePage
	if err := json.NewDecoder(rrList.Body).Decode(&page); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	found := false
	for _, d := range page.Devices {
		if d.ID == created.ID {
			found = true
			if d.DisplayName == nil || *d.DisplayName != "integration-router" {
				t.Fatalf("expected display_name integration-router, got %v", d.DisplayName)
			}
			if d.Metadata == nil || d.Metadata.Location == nil || *d.Metadata.Location != "lab" {
				t.Fatalf("expected metadata.location lab, got %+v", d.Metadata)
			}
			if d.PrimaryIP == nil || *d.PrimaryIP != "192.0.2.20" {
				t.Fatalf("expected primary_ip 192.0.2.20, got %v", d.PrimaryIP)
			}
		}
	}
	if !found {
		t.Fatalf("expected created device %s to appear in list", created.ID)
	}
}

func TestHandler_Postgres_MapL3_DeviceFocus(t *testing.T) {
	adminURL := requireTestDatabaseURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := newTestDatabaseName()
	testDBURL := mustDeriveDatabaseURL(t, adminURL, dbName)

	if err := createDatabase(ctx, adminURL, dbName); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		_ = dropDatabase(context.Background(), adminURL, dbName)
	})

	mConn, err := pgx.Connect(ctx, testDBURL)
	if err != nil {
		t.Fatalf("connect for migrations: %v", err)
	}
	if err := applyMigrations(ctx, mConn, migrationsDir(t)); err != nil {
		_ = mConn.Close(ctx)
		t.Fatalf("apply migrations: %v", err)
	}
	if err := mConn.Close(ctx); err != nil {
		t.Fatalf("close migration connection: %v", err)
	}

	pool, err := db.Open(ctx, testDBURL)
	if err != nil {
		t.Fatalf("open db pool: %v", err)
	}
	t.Cleanup(pool.Close)

	h := NewHandler(NewLogger("error"), pool)
	router := h.Router()

	seedConn, err := pgx.Connect(ctx, testDBURL)
	if err != nil {
		t.Fatalf("connect for seed data: %v", err)
	}
	focusID := "00000000-0000-0000-0000-000000000001"
	peer1ID := "00000000-0000-0000-0000-000000000002"
	peer2ID := "00000000-0000-0000-0000-000000000003"
	multiID := "00000000-0000-0000-0000-000000000004"
	outsideID := "00000000-0000-0000-0000-000000000005"

	if _, err := seedConn.Exec(ctx, `INSERT INTO devices (id, display_name) VALUES ($1::uuid, $2)`, focusID, "focus-device"); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert focus device: %v", err)
	}
	if _, err := seedConn.Exec(ctx, `INSERT INTO devices (id, display_name) VALUES ($1::uuid, $2)`, peer1ID, "peer-1"); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert peer1 device: %v", err)
	}
	if _, err := seedConn.Exec(ctx, `INSERT INTO devices (id, display_name) VALUES ($1::uuid, $2)`, peer2ID, "peer-2"); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert peer2 device: %v", err)
	}
	if _, err := seedConn.Exec(ctx, `INSERT INTO devices (id, display_name) VALUES ($1::uuid, $2)`, multiID, "multi-homed"); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert multi device: %v", err)
	}
	if _, err := seedConn.Exec(ctx, `INSERT INTO devices (id, display_name) VALUES ($1::uuid, $2)`, outsideID, "outside"); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert outside device: %v", err)
	}

	for deviceID, ip := range map[string]string{
		focusID + "|10.0.1.10":  "10.0.1.10",
		focusID + "|10.0.2.10":  "10.0.2.10",
		peer1ID + "|10.0.1.20":  "10.0.1.20",
		peer2ID + "|10.0.2.30":  "10.0.2.30",
		multiID + "|10.0.1.99":  "10.0.1.99",
		multiID + "|10.0.2.99":  "10.0.2.99",
		outsideID + "|10.0.3.5": "10.0.3.5",
	} {
		parts := strings.SplitN(deviceID, "|", 2)
		if len(parts) != 2 {
			_ = seedConn.Close(ctx)
			t.Fatalf("seed key format invalid: %q", deviceID)
		}
		id := parts[0]
		if _, err := seedConn.Exec(ctx, `INSERT INTO ip_addresses (device_id, ip) VALUES ($1::uuid, $2::inet)`, id, ip); err != nil {
			_ = seedConn.Close(ctx)
			t.Fatalf("insert ip seed %s %s: %v", id, ip, err)
		}
	}

	if err := seedConn.Close(ctx); err != nil {
		t.Fatalf("close seed connection: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=device&focusId="+focusID, nil)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var proj mapProjection
	if err := json.NewDecoder(rr.Body).Decode(&proj); err != nil {
		t.Fatalf("decode projection response: %v", err)
	}

	if proj.Layer != "l3" {
		t.Fatalf("expected layer l3, got %q", proj.Layer)
	}
	if proj.Focus == nil || proj.Focus.Type != "device" || proj.Focus.ID != focusID {
		t.Fatalf("expected focus device %s, got %+v", focusID, proj.Focus)
	}

	expectedRegions := []string{"10.0.1.0/24", "10.0.2.0/24"}
	if len(proj.Regions) != len(expectedRegions) {
		t.Fatalf("expected %d regions, got %d", len(expectedRegions), len(proj.Regions))
	}
	for i, want := range expectedRegions {
		if proj.Regions[i].ID != want {
			t.Fatalf("expected region %d id %q, got %q", i, want, proj.Regions[i].ID)
		}
		if proj.Regions[i].Kind != "subnet" {
			t.Fatalf("expected region %d kind subnet, got %q", i, proj.Regions[i].Kind)
		}
	}

	expectedNodeIDs := []string{focusID, peer1ID, peer2ID, multiID}
	if len(proj.Nodes) != len(expectedNodeIDs) {
		t.Fatalf("expected %d nodes, got %d", len(expectedNodeIDs), len(proj.Nodes))
	}
	for i, want := range expectedNodeIDs {
		if proj.Nodes[i].ID != want {
			t.Fatalf("expected node %d id %q, got %q", i, want, proj.Nodes[i].ID)
		}
	}

	nodesByID := make(map[string]mapNode, len(proj.Nodes))
	for _, n := range proj.Nodes {
		nodesByID[n.ID] = n
	}

	focusNode := nodesByID[focusID]
	if len(focusNode.RegionIDs) != 2 || focusNode.RegionIDs[0] != expectedRegions[0] || focusNode.RegionIDs[1] != expectedRegions[1] {
		t.Fatalf("expected focus node region_ids %v, got %v", expectedRegions, focusNode.RegionIDs)
	}
	if focusNode.PrimaryRegionID == nil || *focusNode.PrimaryRegionID != expectedRegions[0] {
		t.Fatalf("expected focus node primary_region_id %q, got %v", expectedRegions[0], focusNode.PrimaryRegionID)
	}

	peer1Node := nodesByID[peer1ID]
	if len(peer1Node.RegionIDs) != 1 || peer1Node.RegionIDs[0] != expectedRegions[0] {
		t.Fatalf("expected peer1 region_ids [%s], got %v", expectedRegions[0], peer1Node.RegionIDs)
	}
	peer2Node := nodesByID[peer2ID]
	if len(peer2Node.RegionIDs) != 1 || peer2Node.RegionIDs[0] != expectedRegions[1] {
		t.Fatalf("expected peer2 region_ids [%s], got %v", expectedRegions[1], peer2Node.RegionIDs)
	}
	multiNode := nodesByID[multiID]
	if len(multiNode.RegionIDs) != 2 || multiNode.RegionIDs[0] != expectedRegions[0] || multiNode.RegionIDs[1] != expectedRegions[1] {
		t.Fatalf("expected multi region_ids %v, got %v", expectedRegions, multiNode.RegionIDs)
	}
	if multiNode.PrimaryRegionID == nil || *multiNode.PrimaryRegionID != expectedRegions[0] {
		t.Fatalf("expected multi primary_region_id %q, got %v", expectedRegions[0], multiNode.PrimaryRegionID)
	}

	if len(proj.Edges) != 3 {
		t.Fatalf("expected 3 edges, got %d", len(proj.Edges))
	}
	for _, e := range proj.Edges {
		if e.Kind != "peer" {
			t.Fatalf("expected edge kind peer, got %q", e.Kind)
		}
		if e.From != focusID {
			t.Fatalf("expected edge from %s, got %s", focusID, e.From)
		}
		if e.To != peer1ID && e.To != peer2ID && e.To != multiID {
			t.Fatalf("unexpected edge to %s", e.To)
		}
	}

	if proj.Truncation.Regions.Returned != 2 || proj.Truncation.Regions.Limit <= 0 || proj.Truncation.Regions.Truncated {
		t.Fatalf("unexpected region truncation: %+v", proj.Truncation.Regions)
	}
	if proj.Truncation.Nodes.Returned != 4 || proj.Truncation.Nodes.Truncated {
		t.Fatalf("unexpected node truncation: %+v", proj.Truncation.Nodes)
	}
	if proj.Truncation.Edges.Returned != 3 || proj.Truncation.Edges.Truncated {
		t.Fatalf("unexpected edge truncation: %+v", proj.Truncation.Edges)
	}
}

func TestHandler_Postgres_MapL3_DeviceFocus_LimitCapsNodes(t *testing.T) {
	adminURL := requireTestDatabaseURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := newTestDatabaseName()
	testDBURL := mustDeriveDatabaseURL(t, adminURL, dbName)

	if err := createDatabase(ctx, adminURL, dbName); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		_ = dropDatabase(context.Background(), adminURL, dbName)
	})

	mConn, err := pgx.Connect(ctx, testDBURL)
	if err != nil {
		t.Fatalf("connect for migrations: %v", err)
	}
	if err := applyMigrations(ctx, mConn, migrationsDir(t)); err != nil {
		_ = mConn.Close(ctx)
		t.Fatalf("apply migrations: %v", err)
	}
	if err := mConn.Close(ctx); err != nil {
		t.Fatalf("close migration connection: %v", err)
	}

	pool, err := db.Open(ctx, testDBURL)
	if err != nil {
		t.Fatalf("open db pool: %v", err)
	}
	t.Cleanup(pool.Close)

	h := NewHandler(NewLogger("error"), pool)
	router := h.Router()

	seedConn, err := pgx.Connect(ctx, testDBURL)
	if err != nil {
		t.Fatalf("connect for seed data: %v", err)
	}
	focusID := "00000000-0000-0000-0000-000000000001"
	peer1ID := "00000000-0000-0000-0000-000000000002"
	peer2ID := "00000000-0000-0000-0000-000000000003"
	multiID := "00000000-0000-0000-0000-000000000004"

	if _, err := seedConn.Exec(ctx, `INSERT INTO devices (id, display_name) VALUES ($1::uuid, $2)`, focusID, "focus-device"); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert focus device: %v", err)
	}
	if _, err := seedConn.Exec(ctx, `INSERT INTO devices (id, display_name) VALUES ($1::uuid, $2)`, peer1ID, "peer-1"); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert peer1 device: %v", err)
	}
	if _, err := seedConn.Exec(ctx, `INSERT INTO devices (id, display_name) VALUES ($1::uuid, $2)`, peer2ID, "peer-2"); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert peer2 device: %v", err)
	}
	if _, err := seedConn.Exec(ctx, `INSERT INTO devices (id, display_name) VALUES ($1::uuid, $2)`, multiID, "multi-homed"); err != nil {
		_ = seedConn.Close(ctx)
		t.Fatalf("insert multi device: %v", err)
	}

	for deviceID, ip := range map[string]string{
		focusID + "|10.0.1.10": "10.0.1.10",
		focusID + "|10.0.2.10": "10.0.2.10",
		peer1ID + "|10.0.1.20": "10.0.1.20",
		peer2ID + "|10.0.2.30": "10.0.2.30",
		multiID + "|10.0.1.99": "10.0.1.99",
		multiID + "|10.0.2.99": "10.0.2.99",
	} {
		parts := strings.SplitN(deviceID, "|", 2)
		if len(parts) != 2 {
			_ = seedConn.Close(ctx)
			t.Fatalf("seed key format invalid: %q", deviceID)
		}
		id := parts[0]
		if _, err := seedConn.Exec(ctx, `INSERT INTO ip_addresses (device_id, ip) VALUES ($1::uuid, $2::inet)`, id, ip); err != nil {
			_ = seedConn.Close(ctx)
			t.Fatalf("insert ip seed %s %s: %v", id, ip, err)
		}
	}

	if err := seedConn.Close(ctx); err != nil {
		t.Fatalf("close seed connection: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=device&focusId="+focusID+"&limit=2", nil)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var proj mapProjection
	if err := json.NewDecoder(rr.Body).Decode(&proj); err != nil {
		t.Fatalf("decode projection response: %v", err)
	}

	if proj.Truncation.Nodes.Limit != 2 {
		t.Fatalf("expected nodes limit 2, got %d", proj.Truncation.Nodes.Limit)
	}
	if proj.Truncation.Nodes.Returned != 2 || !proj.Truncation.Nodes.Truncated {
		t.Fatalf("expected nodes returned=2 truncated=true, got %+v", proj.Truncation.Nodes)
	}

	if len(proj.Nodes) != 2 || proj.Nodes[0].ID != focusID || proj.Nodes[1].ID != peer1ID {
		gotIDs := make([]string, 0, len(proj.Nodes))
		for _, n := range proj.Nodes {
			gotIDs = append(gotIDs, n.ID)
		}
		t.Fatalf("expected nodes [%s %s], got %v", focusID, peer1ID, gotIDs)
	}
}

func TestHandler_Postgres_MapL3_SubnetFocus(t *testing.T) {
	adminURL := requireTestDatabaseURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := newTestDatabaseName()
	testDBURL := mustDeriveDatabaseURL(t, adminURL, dbName)

	if err := createDatabase(ctx, adminURL, dbName); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		_ = dropDatabase(context.Background(), adminURL, dbName)
	})

	mConn, err := pgx.Connect(ctx, testDBURL)
	if err != nil {
		t.Fatalf("connect for migrations: %v", err)
	}
	if err := applyMigrations(ctx, mConn, migrationsDir(t)); err != nil {
		_ = mConn.Close(ctx)
		t.Fatalf("apply migrations: %v", err)
	}
	if err := mConn.Close(ctx); err != nil {
		t.Fatalf("close migration connection: %v", err)
	}

	pool, err := db.Open(ctx, testDBURL)
	if err != nil {
		t.Fatalf("open db pool: %v", err)
	}
	t.Cleanup(pool.Close)

	h := NewHandler(NewLogger("error"), pool)
	router := h.Router()

	seedConn, err := pgx.Connect(ctx, testDBURL)
	if err != nil {
		t.Fatalf("connect for seed data: %v", err)
	}

	device1ID := "00000000-0000-0000-0000-000000000001"
	device2ID := "00000000-0000-0000-0000-000000000002"
	device3ID := "00000000-0000-0000-0000-000000000003"
	outsideID := "00000000-0000-0000-0000-000000000004"

	for id, name := range map[string]string{
		device1ID: "subnet-device-1",
		device2ID: "subnet-device-2",
		device3ID: "subnet-device-3",
		outsideID: "outside-subnet",
	} {
		if _, err := seedConn.Exec(ctx, `INSERT INTO devices (id, display_name) VALUES ($1::uuid, $2)`, id, name); err != nil {
			_ = seedConn.Close(ctx)
			t.Fatalf("insert device %s: %v", id, err)
		}
	}

	for deviceID, ip := range map[string]string{
		device1ID: "10.0.1.10",
		device2ID: "10.0.1.20",
		device3ID: "10.0.1.30",
		outsideID: "10.0.2.10",
	} {
		if _, err := seedConn.Exec(ctx, `INSERT INTO ip_addresses (device_id, ip) VALUES ($1::uuid, $2::inet)`, deviceID, ip); err != nil {
			_ = seedConn.Close(ctx)
			t.Fatalf("insert ip seed %s %s: %v", deviceID, ip, err)
		}
	}

	if err := seedConn.Close(ctx); err != nil {
		t.Fatalf("close seed connection: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=subnet&focusId=10.0.1.5/24", nil)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var proj mapProjection
	if err := json.NewDecoder(rr.Body).Decode(&proj); err != nil {
		t.Fatalf("decode projection response: %v", err)
	}

	if proj.Layer != "l3" {
		t.Fatalf("expected layer l3, got %q", proj.Layer)
	}
	if proj.Focus == nil || proj.Focus.Type != "subnet" || proj.Focus.ID != "10.0.1.0/24" {
		t.Fatalf("expected focus subnet 10.0.1.0/24, got %+v", proj.Focus)
	}

	if len(proj.Regions) != 1 || proj.Regions[0].ID != "10.0.1.0/24" || proj.Regions[0].Kind != "subnet" {
		t.Fatalf("expected single subnet region 10.0.1.0/24, got %+v", proj.Regions)
	}

	expectedNodeIDs := []string{device1ID, device2ID, device3ID}
	if len(proj.Nodes) != len(expectedNodeIDs) {
		t.Fatalf("expected %d nodes, got %d", len(expectedNodeIDs), len(proj.Nodes))
	}
	for i, want := range expectedNodeIDs {
		if proj.Nodes[i].ID != want {
			t.Fatalf("expected node %d id %q, got %q", i, want, proj.Nodes[i].ID)
		}
		if len(proj.Nodes[i].RegionIDs) != 1 || proj.Nodes[i].RegionIDs[0] != "10.0.1.0/24" {
			t.Fatalf("expected node %s region_ids [10.0.1.0/24], got %v", proj.Nodes[i].ID, proj.Nodes[i].RegionIDs)
		}
		if proj.Nodes[i].PrimaryRegionID == nil || *proj.Nodes[i].PrimaryRegionID != "10.0.1.0/24" {
			t.Fatalf("expected node %s primary_region_id 10.0.1.0/24, got %v", proj.Nodes[i].ID, proj.Nodes[i].PrimaryRegionID)
		}
	}

	if len(proj.Edges) != 0 {
		t.Fatalf("expected no edges, got %d", len(proj.Edges))
	}

	rrLimited := httptest.NewRecorder()
	reqLimited := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=subnet&focusId=10.0.1.0/24&limit=2", nil)
	router.ServeHTTP(rrLimited, reqLimited)

	if rrLimited.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rrLimited.Code, rrLimited.Body.String())
	}

	var projLimited mapProjection
	if err := json.NewDecoder(rrLimited.Body).Decode(&projLimited); err != nil {
		t.Fatalf("decode projection response (limited): %v", err)
	}
	if projLimited.Truncation.Nodes.Limit != 2 {
		t.Fatalf("expected nodes limit 2, got %d", projLimited.Truncation.Nodes.Limit)
	}
	if projLimited.Truncation.Nodes.Returned != 2 || !projLimited.Truncation.Nodes.Truncated {
		t.Fatalf("expected nodes returned=2 truncated=true, got %+v", projLimited.Truncation.Nodes)
	}
	if len(projLimited.Nodes) != 2 || projLimited.Nodes[0].ID != device1ID || projLimited.Nodes[1].ID != device2ID {
		gotIDs := make([]string, 0, len(projLimited.Nodes))
		for _, n := range projLimited.Nodes {
			gotIDs = append(gotIDs, n.ID)
		}
		t.Fatalf("expected nodes [%s %s], got %v", device1ID, device2ID, gotIDs)
	}
}

func TestHandler_Postgres_MapL3_SubnetFocus_EmptySubnet_Returns200(t *testing.T) {
	adminURL := requireTestDatabaseURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbName := newTestDatabaseName()
	testDBURL := mustDeriveDatabaseURL(t, adminURL, dbName)

	if err := createDatabase(ctx, adminURL, dbName); err != nil {
		t.Fatalf("create database: %v", err)
	}
	t.Cleanup(func() {
		_ = dropDatabase(context.Background(), adminURL, dbName)
	})

	mConn, err := pgx.Connect(ctx, testDBURL)
	if err != nil {
		t.Fatalf("connect for migrations: %v", err)
	}
	if err := applyMigrations(ctx, mConn, migrationsDir(t)); err != nil {
		_ = mConn.Close(ctx)
		t.Fatalf("apply migrations: %v", err)
	}
	if err := mConn.Close(ctx); err != nil {
		t.Fatalf("close migration connection: %v", err)
	}

	pool, err := db.Open(ctx, testDBURL)
	if err != nil {
		t.Fatalf("open db pool: %v", err)
	}
	t.Cleanup(pool.Close)

	h := NewHandler(NewLogger("error"), pool)
	router := h.Router()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/map/l3?focusType=subnet&focusId=10.99.0.0/16", nil)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var proj mapProjection
	if err := json.NewDecoder(rr.Body).Decode(&proj); err != nil {
		t.Fatalf("decode projection response: %v", err)
	}

	if proj.Focus == nil || proj.Focus.Type != "subnet" || proj.Focus.ID != "10.99.0.0/16" {
		t.Fatalf("expected focus subnet 10.99.0.0/16, got %+v", proj.Focus)
	}
	if len(proj.Regions) != 1 || proj.Regions[0].ID != "10.99.0.0/16" {
		t.Fatalf("expected single region 10.99.0.0/16, got %+v", proj.Regions)
	}
	if len(proj.Nodes) != 0 {
		t.Fatalf("expected 0 nodes, got %d", len(proj.Nodes))
	}
	if len(proj.Edges) != 0 {
		t.Fatalf("expected 0 edges, got %d", len(proj.Edges))
	}
}
