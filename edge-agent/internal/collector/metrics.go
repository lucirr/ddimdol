package collector

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// Metrics holds a point-in-time snapshot of host resource utilisation.
type Metrics struct {
	CPUPct    float64   `json:"cpu_pct"`
	MemPct    float64   `json:"mem_pct"`
	DiskPct   float64   `json:"disk_pct"`
	Timestamp time.Time `json:"timestamp"`
}

// Collect gathers CPU, memory, and disk metrics from the host using gopsutil.
// The CPU measurement blocks for one second to obtain a meaningful sample.
func Collect() (*Metrics, error) {
	cpuPercents, err := cpu.Percent(time.Second, false)
	if err != nil {
		return nil, fmt.Errorf("cpu.Percent: %w", err)
	}
	if len(cpuPercents) == 0 {
		return nil, fmt.Errorf("cpu.Percent returned empty slice")
	}

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("mem.VirtualMemory: %w", err)
	}

	diskStat, err := disk.Usage("/")
	if err != nil {
		return nil, fmt.Errorf("disk.Usage: %w", err)
	}

	return &Metrics{
		CPUPct:    cpuPercents[0],
		MemPct:    vmStat.UsedPercent,
		DiskPct:   diskStat.UsedPercent,
		Timestamp: time.Now().UTC(),
	}, nil
}
