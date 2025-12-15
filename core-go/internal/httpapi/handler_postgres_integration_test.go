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
		}
	}
	if !found {
		t.Fatalf("expected created device %s to appear in list", created.ID)
	}
}
