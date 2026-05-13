package collector

import (
	"testing"
	"time"
)

func TestCollect_ReturnsValidMetrics(t *testing.T) {
	m, err := Collect()
	if err != nil {
		// Some platforms (e.g. darwin without CGO) do not implement all gopsutil
		// backends. Skip rather than fail so CI on those hosts still passes.
		t.Skipf("Collect() not supported on this platform: %v", err)
	}

	if m == nil {
		t.Fatal("Collect() returned nil metrics")
	}

	if m.CPUPct < 0 || m.CPUPct > 100 {
		t.Errorf("CPUPct out of range [0,100]: %f", m.CPUPct)
	}
	if m.MemPct < 0 || m.MemPct > 100 {
		t.Errorf("MemPct out of range [0,100]: %f", m.MemPct)
	}
	if m.DiskPct < 0 || m.DiskPct > 100 {
		t.Errorf("DiskPct out of range [0,100]: %f", m.DiskPct)
	}

	// Timestamp should be recent (within last 5 seconds).
	age := time.Since(m.Timestamp)
	if age < 0 || age > 5*time.Second {
		t.Errorf("Timestamp looks stale: %v ago", age)
	}
}

func TestCollect_TimestampIsUTC(t *testing.T) {
	m, err := Collect()
	if err != nil {
		t.Skipf("Collect() not supported on this platform: %v", err)
	}

	if m.Timestamp.Location() != time.UTC {
		t.Errorf("Timestamp location: got %v, want UTC", m.Timestamp.Location())
	}
}
