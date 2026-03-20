package agent

import (
	"time"

	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
)

// CollectMetrics gathers system metrics using gopsutil
func CollectMetrics(serverID string) (*models.ServerMetrics, error) {
	parsedUUID, err := uuid.Parse(serverID)
	if err != nil {
		return nil, err
	}

	metrics := &models.ServerMetrics{
		ServerID:   parsedUUID,
		RecordedAt: time.Now(),
	}

	// CPU
	cpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercents) > 0 {
		metrics.CPUPercent = cpuPercents[0]
	}

	// Memory
	vMem, err := mem.VirtualMemory()
	if err == nil {
		metrics.RAMTotalMB = int64(vMem.Total / 1024 / 1024)
		metrics.RAMUsedMB = int64(vMem.Used / 1024 / 1024)
	}

	// Disk
	dInfo, err := disk.Usage("/")
	if err == nil {
		metrics.DiskTotalGB = float64(dInfo.Total) / 1024 / 1024 / 1024
		metrics.DiskUsedGB = float64(dInfo.Used) / 1024 / 1024 / 1024
	}

	// Load Average
	lInfo, err := load.Avg()
	if err == nil {
		metrics.LoadAvg1 = lInfo.Load1
		metrics.LoadAvg5 = lInfo.Load5
		metrics.LoadAvg15 = lInfo.Load15
	}

	// Network
	nInfo, err := net.IOCounters(false)
	if err == nil && len(nInfo) > 0 {
		metrics.NetworkRxBytes = int64(nInfo[0].BytesRecv)
		metrics.NetworkTxBytes = int64(nInfo[0].BytesSent)
	}

	return metrics, nil
}
