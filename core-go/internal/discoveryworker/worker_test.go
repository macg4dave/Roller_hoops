package discoveryworker

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"roller_hoops/core-go/internal/sqlcgen"
)

type fakeQueries struct {
	claimFn               func(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error)
	updateFn              func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error)
	insertFn              func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error
	createFn              func(ctx context.Context, displayName *string) (sqlcgen.Device, error)
	findByMacFn           func(ctx context.Context, mac string) (string, error)
	findByIPFn            func(ctx context.Context, ip string) (string, error)
	upsertIPFn            func(ctx context.Context, arg sqlcgen.UpsertDeviceIPParams) error
	upsertMACFn           func(ctx context.Context, arg sqlcgen.UpsertDeviceMACParams) error
	insertIPObs           func(ctx context.Context, arg sqlcgen.InsertIPObservationParams) error
	insertMACObs          func(ctx context.Context, arg sqlcgen.InsertMACObservationParams) error
	insertNameCandidateFn func(ctx context.Context, arg sqlcgen.InsertDeviceNameCandidateParams) error
	setDisplayNameFn      func(ctx context.Context, arg sqlcgen.SetDeviceDisplayNameIfUnsetParams) (int64, error)
	upsertSNMPFn          func(ctx context.Context, arg sqlcgen.UpsertDeviceSNMPParams) error
	upsertInterfaceFn     func(ctx context.Context, arg sqlcgen.UpsertInterfaceFromSNMPParams) (string, error)
	upsertIfaceByNameFn   func(ctx context.Context, arg sqlcgen.UpsertInterfaceByNameParams) (string, error)
	upsertInterfaceMacFn  func(ctx context.Context, arg sqlcgen.UpsertInterfaceMACParams) error
	linkMacFn             func(ctx context.Context, arg sqlcgen.LinkDeviceMACToInterfaceParams) (int64, error)
	upsertVlanFn          func(ctx context.Context, arg sqlcgen.UpsertInterfaceVLANParams) error
	upsertLinkFn          func(ctx context.Context, arg sqlcgen.UpsertLinkParams) error
	upsertServiceFn       func(ctx context.Context, arg sqlcgen.UpsertServiceFromScanParams) error
}

func (f *fakeQueries) ClaimNextDiscoveryRun(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error) {
	return f.claimFn(ctx, stats)
}

func (f *fakeQueries) UpdateDiscoveryRun(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
	return f.updateFn(ctx, arg)
}

func (f *fakeQueries) InsertDiscoveryRunLog(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error {
	return f.insertFn(ctx, arg)
}

func (f *fakeQueries) CreateDevice(ctx context.Context, displayName *string) (sqlcgen.Device, error) {
	if f.createFn == nil {
		return sqlcgen.Device{}, nil
	}
	return f.createFn(ctx, displayName)
}

func (f *fakeQueries) FindDeviceIDByMAC(ctx context.Context, mac string) (string, error) {
	if f.findByMacFn == nil {
		return "", pgx.ErrNoRows
	}
	return f.findByMacFn(ctx, mac)
}

func (f *fakeQueries) FindDeviceIDByIP(ctx context.Context, ip string) (string, error) {
	if f.findByIPFn == nil {
		return "", pgx.ErrNoRows
	}
	return f.findByIPFn(ctx, ip)
}

func (f *fakeQueries) UpsertDeviceIP(ctx context.Context, arg sqlcgen.UpsertDeviceIPParams) error {
	if f.upsertIPFn == nil {
		return nil
	}
	return f.upsertIPFn(ctx, arg)
}

func (f *fakeQueries) UpsertDeviceMAC(ctx context.Context, arg sqlcgen.UpsertDeviceMACParams) error {
	if f.upsertMACFn == nil {
		return nil
	}
	return f.upsertMACFn(ctx, arg)
}

func (f *fakeQueries) InsertIPObservation(ctx context.Context, arg sqlcgen.InsertIPObservationParams) error {
	if f.insertIPObs == nil {
		return nil
	}
	return f.insertIPObs(ctx, arg)
}

func (f *fakeQueries) InsertMACObservation(ctx context.Context, arg sqlcgen.InsertMACObservationParams) error {
	if f.insertMACObs == nil {
		return nil
	}
	return f.insertMACObs(ctx, arg)
}

func (f *fakeQueries) InsertDeviceNameCandidate(ctx context.Context, arg sqlcgen.InsertDeviceNameCandidateParams) error {
	if f.insertNameCandidateFn == nil {
		return nil
	}
	return f.insertNameCandidateFn(ctx, arg)
}

func (f *fakeQueries) SetDeviceDisplayNameIfUnset(ctx context.Context, arg sqlcgen.SetDeviceDisplayNameIfUnsetParams) (int64, error) {
	if f.setDisplayNameFn == nil {
		return 0, nil
	}
	return f.setDisplayNameFn(ctx, arg)
}

func (f *fakeQueries) UpsertDeviceSNMP(ctx context.Context, arg sqlcgen.UpsertDeviceSNMPParams) error {
	if f.upsertSNMPFn == nil {
		return nil
	}
	return f.upsertSNMPFn(ctx, arg)
}

func (f *fakeQueries) UpsertInterfaceFromSNMP(ctx context.Context, arg sqlcgen.UpsertInterfaceFromSNMPParams) (string, error) {
	if f.upsertInterfaceFn == nil {
		return "", nil
	}
	return f.upsertInterfaceFn(ctx, arg)
}

func (f *fakeQueries) UpsertInterfaceByName(ctx context.Context, arg sqlcgen.UpsertInterfaceByNameParams) (string, error) {
	if f.upsertIfaceByNameFn == nil {
		return "", nil
	}
	return f.upsertIfaceByNameFn(ctx, arg)
}

func (f *fakeQueries) UpsertInterfaceMAC(ctx context.Context, arg sqlcgen.UpsertInterfaceMACParams) error {
	if f.upsertInterfaceMacFn == nil {
		return nil
	}
	return f.upsertInterfaceMacFn(ctx, arg)
}

func (f *fakeQueries) LinkDeviceMACToInterface(ctx context.Context, arg sqlcgen.LinkDeviceMACToInterfaceParams) (int64, error) {
	if f.linkMacFn == nil {
		return 0, nil
	}
	return f.linkMacFn(ctx, arg)
}

func (f *fakeQueries) UpsertInterfaceVLAN(ctx context.Context, arg sqlcgen.UpsertInterfaceVLANParams) error {
	if f.upsertVlanFn == nil {
		return nil
	}
	return f.upsertVlanFn(ctx, arg)
}

func (f *fakeQueries) UpsertLink(ctx context.Context, arg sqlcgen.UpsertLinkParams) error {
	if f.upsertLinkFn == nil {
		return nil
	}
	return f.upsertLinkFn(ctx, arg)
}

func (f *fakeQueries) UpsertServiceFromScan(ctx context.Context, arg sqlcgen.UpsertServiceFromScanParams) error {
	if f.upsertServiceFn == nil {
		return nil
	}
	return f.upsertServiceFn(ctx, arg)
}

func writeTempARPFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "arp-*.txt")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(f.Name()) })
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		t.Fatalf("write temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}
	return f.Name()
}

func TestWorker_RunOnce_NoQueuedRuns(t *testing.T) {
	q := &fakeQueries{
		claimFn: func(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error) {
			return sqlcgen.DiscoveryRun{}, pgx.ErrNoRows
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			t.Fatalf("UpdateDiscoveryRun should not be called")
			return sqlcgen.DiscoveryRun{}, nil
		},
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error {
			t.Fatalf("InsertDiscoveryRunLog should not be called")
			return nil
		},
	}

	arpPath := writeTempARPFile(t, "IP address       HW type     Flags       HW address            Mask     Device\n")
	w := New(zerolog.Nop(), q, Options{PollInterval: 0, RunDelay: 0, ARPTablePath: arpPath}, nil)
	processed, err := w.runOnce(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if processed {
		t.Fatalf("expected processed=false")
	}
}

func TestWorker_RunOnce_ClaimsAndCompletes(t *testing.T) {
	var (
		seenStarted   bool
		seenCompleted bool
		updatedStatus string
	)

	now := time.Now()
	q := &fakeQueries{
		claimFn: func(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error) {
			if stats == nil || stats["stage"] != "running" {
				t.Fatalf("expected running stats, got %#v", stats)
			}
			return sqlcgen.DiscoveryRun{ID: "run-1", Status: "running", StartedAt: now}, nil
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			updatedStatus = arg.Status
			if arg.Status != "succeeded" {
				t.Fatalf("expected succeeded, got %q", arg.Status)
			}
			if arg.CompletedAt == nil {
				t.Fatalf("expected completed_at set")
			}
			return sqlcgen.DiscoveryRun{ID: arg.ID, Status: arg.Status, StartedAt: now, CompletedAt: arg.CompletedAt}, nil
		},
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error {
			switch arg.Message {
			case "discovery run started":
				seenStarted = true
			case "discovery run completed":
				seenCompleted = true
			}
			return nil
		},
	}

	arpPath := writeTempARPFile(t, "IP address       HW type     Flags       HW address            Mask     Device\n")
	w := New(zerolog.Nop(), q, Options{PollInterval: 0, RunDelay: 0, ARPTablePath: arpPath}, nil)
	processed, err := w.runOnce(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !processed {
		t.Fatalf("expected processed=true")
	}
	if updatedStatus != "succeeded" {
		t.Fatalf("expected run to succeed, got %q", updatedStatus)
	}
	if !seenStarted || !seenCompleted {
		t.Fatalf("expected both logs, got started=%v completed=%v", seenStarted, seenCompleted)
	}
}

func TestWorker_RunOnce_FailsRunWhenUpdateFails(t *testing.T) {
	q := &fakeQueries{}

	q.claimFn = func(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error) {
		return sqlcgen.DiscoveryRun{ID: "run-2", Status: "running"}, nil
	}

	updateCalls := 0
	q.updateFn = func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
		updateCalls++
		if updateCalls == 1 {
			return sqlcgen.DiscoveryRun{}, errors.New("boom")
		}
		if arg.Status != "failed" {
			t.Fatalf("expected failed status on retry, got %q", arg.Status)
		}
		if arg.LastError == nil || *arg.LastError == "" {
			t.Fatalf("expected last_error to be set")
		}
		return sqlcgen.DiscoveryRun{ID: arg.ID, Status: arg.Status}, nil
	}

	q.insertFn = func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error {
		return nil
	}

	arpPath := writeTempARPFile(t, "IP address       HW type     Flags       HW address            Mask     Device\n")
	w := New(zerolog.Nop(), q, Options{PollInterval: 0, RunDelay: 0, ARPTablePath: arpPath}, nil)
	processed, err := w.runOnce(context.Background())
	if !processed {
		t.Fatalf("expected processed=true")
	}
	if err == nil {
		t.Fatalf("expected error")
	}
	if updateCalls < 2 {
		t.Fatalf("expected at least two update calls, got %d", updateCalls)
	}
}

func TestParseProcNetARP_SkipsIncompleteRows(t *testing.T) {
	entries, err := parseProcNetARP("IP address       HW type     Flags       HW address            Mask     Device\n")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestWorker_RunOnce_ARPEntryCreatesAndUpserts(t *testing.T) {
	arpPath := writeTempARPFile(t, "IP address       HW type     Flags       HW address            Mask     Device\n192.168.1.10      0x1         0x2         aa:bb:cc:dd:ee:ff     *        eth0\n")

	var createdDevices int
	var upsertIPs int
	var upsertMACs int
	var ipObs int
	var macObs int

	q := &fakeQueries{
		claimFn: func(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error) {
			return sqlcgen.DiscoveryRun{ID: "run-arp", Status: "running"}, nil
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			if arg.Status != "succeeded" {
				t.Fatalf("expected succeeded, got %q", arg.Status)
			}
			return sqlcgen.DiscoveryRun{ID: arg.ID, Status: arg.Status}, nil
		},
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error {
			return nil
		},
		findByMacFn: func(ctx context.Context, mac string) (string, error) {
			return "", pgx.ErrNoRows
		},
		createFn: func(ctx context.Context, displayName *string) (sqlcgen.Device, error) {
			createdDevices++
			return sqlcgen.Device{ID: "dev-1"}, nil
		},
		upsertIPFn: func(ctx context.Context, arg sqlcgen.UpsertDeviceIPParams) error {
			upsertIPs++
			if arg.DeviceID != "dev-1" || arg.IP != "192.168.1.10" {
				t.Fatalf("unexpected ip upsert: %#v", arg)
			}
			return nil
		},
		upsertMACFn: func(ctx context.Context, arg sqlcgen.UpsertDeviceMACParams) error {
			upsertMACs++
			if arg.DeviceID != "dev-1" || arg.MAC != "aa:bb:cc:dd:ee:ff" {
				t.Fatalf("unexpected mac upsert: %#v", arg)
			}
			return nil
		},
		insertIPObs: func(ctx context.Context, arg sqlcgen.InsertIPObservationParams) error {
			ipObs++
			if arg.RunID != "run-arp" || arg.DeviceID != "dev-1" || arg.IP != "192.168.1.10" {
				t.Fatalf("unexpected ip observation: %#v", arg)
			}
			return nil
		},
		insertMACObs: func(ctx context.Context, arg sqlcgen.InsertMACObservationParams) error {
			macObs++
			if arg.RunID != "run-arp" || arg.DeviceID != "dev-1" || arg.MAC != "aa:bb:cc:dd:ee:ff" {
				t.Fatalf("unexpected mac observation: %#v", arg)
			}
			return nil
		},
	}

	w := New(zerolog.Nop(), q, Options{PollInterval: 0, RunDelay: 0, ARPTablePath: arpPath}, nil)
	processed, err := w.runOnce(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !processed {
		t.Fatalf("expected processed=true")
	}
	if createdDevices != 1 || upsertIPs != 1 || upsertMACs != 1 || ipObs != 1 || macObs != 1 {
		t.Fatalf(
			"expected 1 create + 1 ip upsert + 1 mac upsert + 1 ip obs + 1 mac obs, got create=%d ip=%d mac=%d ipObs=%d macObs=%d",
			createdDevices,
			upsertIPs,
			upsertMACs,
			ipObs,
			macObs,
		)
	}
}

func TestWorker_RunOnce_ARPEntryReusesDeviceByIP(t *testing.T) {
	arpPath := writeTempARPFile(t, "IP address       HW type     Flags       HW address            Mask     Device\n10.0.0.5          0x1         0x2         aa:bb:cc:dd:ee:11     *        eth0\n")

	var createdDevices int
	var upsertIPs int
	var upsertMACs int

	q := &fakeQueries{
		claimFn: func(ctx context.Context, stats map[string]any) (sqlcgen.DiscoveryRun, error) {
			return sqlcgen.DiscoveryRun{ID: "run-arp", Status: "running"}, nil
		},
		updateFn: func(ctx context.Context, arg sqlcgen.UpdateDiscoveryRunParams) (sqlcgen.DiscoveryRun, error) {
			return sqlcgen.DiscoveryRun{ID: arg.ID, Status: arg.Status}, nil
		},
		insertFn: func(ctx context.Context, arg sqlcgen.InsertDiscoveryRunLogParams) error { return nil },
		findByMacFn: func(ctx context.Context, mac string) (string, error) {
			return "", pgx.ErrNoRows
		},
		findByIPFn: func(ctx context.Context, ip string) (string, error) {
			if ip != "10.0.0.5" {
				t.Fatalf("unexpected ip lookup: %q", ip)
			}
			return "dev-existing", nil
		},
		createFn: func(ctx context.Context, displayName *string) (sqlcgen.Device, error) {
			createdDevices++
			return sqlcgen.Device{ID: "dev-new"}, nil
		},
		upsertIPFn: func(ctx context.Context, arg sqlcgen.UpsertDeviceIPParams) error {
			upsertIPs++
			if arg.DeviceID != "dev-existing" {
				t.Fatalf("expected existing device id, got %#v", arg)
			}
			return nil
		},
		upsertMACFn: func(ctx context.Context, arg sqlcgen.UpsertDeviceMACParams) error {
			upsertMACs++
			if arg.DeviceID != "dev-existing" {
				t.Fatalf("expected existing device id, got %#v", arg)
			}
			return nil
		},
	}

	w := New(zerolog.Nop(), q, Options{PollInterval: 0, RunDelay: 0, ARPTablePath: arpPath}, nil)
	processed, err := w.runOnce(context.Background())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !processed {
		t.Fatalf("expected processed=true")
	}
	if createdDevices != 0 {
		t.Fatalf("expected no device creation, got %d", createdDevices)
	}
	if upsertIPs != 1 || upsertMACs != 1 {
		t.Fatalf("expected 1 ip upsert + 1 mac upsert, got ip=%d mac=%d", upsertIPs, upsertMACs)
	}
}
