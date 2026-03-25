package collector

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"

	"github.com/tudorAbrudan/tracelog/internal/models"
)

type SystemCollector struct {
	prevNetRx uint64
	prevNetTx uint64
	prevDiskR uint64
	prevDiskW uint64
	firstRun  bool
}

func NewSystemCollector() *SystemCollector {
	return &SystemCollector{firstRun: true}
}

func (c *SystemCollector) Collect(ctx context.Context) (*models.SystemMetrics, error) {
	m := &models.SystemMetrics{
		Ts: time.Now().UTC().Truncate(time.Second),
	}

	if cpuPercent, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(cpuPercent) > 0 {
		m.CPUPercent = cpuPercent[0]
	}

	if vmem, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		m.MemUsed = vmem.Used
		m.MemTotal = vmem.Total
	}

	if swap, err := mem.SwapMemoryWithContext(ctx); err == nil {
		m.SwapUsed = swap.Used
		m.SwapTotal = swap.Total
	}

	if usage, err := disk.UsageWithContext(ctx, "/"); err == nil {
		m.DiskUsed = usage.Used
		m.DiskTotal = usage.Total
	}

	if ioCounters, err := disk.IOCountersWithContext(ctx); err == nil {
		var totalR, totalW uint64
		for _, io := range ioCounters {
			totalR += io.ReadBytes
			totalW += io.WriteBytes
		}
		if !c.firstRun {
			m.DiskReadBytes = totalR - c.prevDiskR
			m.DiskWriteBytes = totalW - c.prevDiskW
		}
		c.prevDiskR = totalR
		c.prevDiskW = totalW
	}

	if netCounters, err := net.IOCountersWithContext(ctx, false); err == nil && len(netCounters) > 0 {
		totalRx := netCounters[0].BytesRecv
		totalTx := netCounters[0].BytesSent
		if !c.firstRun {
			m.NetRxBytes = totalRx - c.prevNetRx
			m.NetTxBytes = totalTx - c.prevNetTx
		}
		c.prevNetRx = totalRx
		c.prevNetTx = totalTx
	}

	if avg, err := load.AvgWithContext(ctx); err == nil {
		m.Load1 = avg.Load1
		m.Load5 = avg.Load5
		m.Load15 = avg.Load15
	}

	if info, err := host.UptimeWithContext(ctx); err == nil {
		m.Uptime = info
	}

	c.firstRun = false
	return m, nil
}
