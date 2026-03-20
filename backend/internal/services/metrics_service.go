package services

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HostMetrics holds a snapshot of system resource usage.
type HostMetrics struct {
	CPUPercent   float64 `json:"cpu_percent"`
	RAMUsedMB    int64   `json:"ram_used_mb"`
	RAMTotalMB   int64   `json:"ram_total_mb"`
	RAMPercent   float64 `json:"ram_percent"`
	DiskUsedGB   float64 `json:"disk_used_gb"`
	DiskTotalGB  float64 `json:"disk_total_gb"`
	DiskPercent  float64 `json:"disk_percent"`
	LoadAvg1     float64 `json:"load_avg_1"`
	LoadAvg5     float64 `json:"load_avg_5"`
	LoadAvg15    float64 `json:"load_avg_15"`
	NetworkRxMB  float64 `json:"network_rx_mb"`
	NetworkTxMB  float64 `json:"network_tx_mb"`
	Uptime       string  `json:"uptime"`
	ProcessCount int     `json:"process_count"`
}

// ServiceStatus represents the health of a discovered service.
type ServiceStatus struct {
	Name   string `json:"name"`
	Engine string `json:"engine"`
	Type   string `json:"type"`
	Status string `json:"status"` // "running", "stopped", "unknown"
	Port   int    `json:"port,omitempty"`
}

type MetricsService struct {
	pool         *pgxpool.Pool
	prevCPUIdle  uint64
	prevCPUTotal uint64
}

func NewMetricsService(pool *pgxpool.Pool) *MetricsService {
	return &MetricsService{pool: pool}
}

// CollectHostMetrics gathers system metrics from /proc.
func (s *MetricsService) CollectHostMetrics() (*HostMetrics, error) {
	m := &HostMetrics{}

	// CPU
	m.CPUPercent = s.readCPUPercent()

	// Memory
	s.readMemInfo(m)

	// Disk
	s.readDiskUsage(m)

	// Load average
	s.readLoadAvg(m)

	// Network
	s.readNetwork(m)

	// Uptime
	m.Uptime = s.readUptime()

	// Process count
	m.ProcessCount = s.readProcessCount()

	return m, nil
}

// SaveMetrics persists a metrics snapshot to the database.
func (s *MetricsService) SaveMetrics(ctx context.Context, serverID string, m *HostMetrics) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO server_metrics (server_id, cpu_percent, ram_used_mb, ram_total_mb,
		 disk_used_gb, disk_total_gb, load_avg_1, load_avg_5, load_avg_15,
		 network_rx_bytes, network_tx_bytes)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		serverID, m.CPUPercent, m.RAMUsedMB, m.RAMTotalMB,
		m.DiskUsedGB, m.DiskTotalGB, m.LoadAvg1, m.LoadAvg5, m.LoadAvg15,
		int64(m.NetworkRxMB*1024*1024), int64(m.NetworkTxMB*1024*1024),
	)
	return err
}

// GetHistory returns historical metrics for a server within a time range.
func (s *MetricsService) GetHistory(ctx context.Context, serverID string, hours int) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT cpu_percent, ram_used_mb, ram_total_mb, disk_used_gb, disk_total_gb,
		 load_avg_1, load_avg_5, load_avg_15, network_rx_bytes, network_tx_bytes, recorded_at
		 FROM server_metrics WHERE server_id = $1::uuid AND recorded_at > NOW() - INTERVAL '1 hour' * $2
		 ORDER BY recorded_at ASC LIMIT 500`,
		serverID, hours,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var cpuPct float64
		var ramUsed, ramTotal int64
		var diskUsed, diskTotal, load1, load5, load15 float64
		var netRx, netTx int64
		var recordedAt time.Time
		if err := rows.Scan(&cpuPct, &ramUsed, &ramTotal, &diskUsed, &diskTotal,
			&load1, &load5, &load15, &netRx, &netTx, &recordedAt); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"cpu_percent":  cpuPct,
			"ram_used_mb":  ramUsed,
			"ram_total_mb": ramTotal,
			"disk_used_gb": diskUsed,
			"disk_total_gb": diskTotal,
			"load_avg_1":   load1,
			"load_avg_5":   load5,
			"load_avg_15":  load15,
			"network_rx_mb": float64(netRx) / 1024 / 1024,
			"network_tx_mb": float64(netTx) / 1024 / 1024,
			"recorded_at":  recordedAt,
		})
	}
	return results, nil
}

// GetServiceStatuses checks known services for responsiveness.
func (s *MetricsService) GetServiceStatuses() []ServiceStatus {
	type probe struct {
		Name   string
		Engine string
		Type   string
		Ports  []int
		Hosts  []string
	}

	probes := []probe{
		{Name: "PostgreSQL", Engine: "postgresql", Type: "database", Ports: []int{5432}, Hosts: []string{"postgres", "localhost"}},
		{Name: "MySQL/MariaDB", Engine: "mysql", Type: "database", Ports: []int{3306}, Hosts: []string{"localhost"}},
		{Name: "MongoDB", Engine: "mongodb", Type: "database", Ports: []int{27017}, Hosts: []string{"localhost"}},
		{Name: "Redis", Engine: "redis", Type: "cache", Ports: []int{6379}, Hosts: []string{"redis", "localhost"}},
		{Name: "Memcached", Engine: "memcached", Type: "cache", Ports: []int{11211}, Hosts: []string{"localhost"}},
		{Name: "Nginx", Engine: "nginx", Type: "webserver", Ports: []int{80, 8080}, Hosts: []string{"localhost"}},
		{Name: "Apache", Engine: "apache", Type: "webserver", Ports: []int{80, 8080}, Hosts: []string{"localhost"}},
		{Name: "Elasticsearch", Engine: "elasticsearch", Type: "search", Ports: []int{9200}, Hosts: []string{"localhost"}},
		{Name: "RabbitMQ", Engine: "rabbitmq", Type: "queue", Ports: []int{5672}, Hosts: []string{"localhost"}},
		{Name: "SSH", Engine: "ssh", Type: "app", Ports: []int{22}, Hosts: []string{"localhost"}},
	}

	// Also get the gateway for checking host services
	gateway := getGatewayIP()

	var statuses []ServiceStatus
	for _, p := range probes {
		hosts := append(p.Hosts, gateway)
		status := "stopped"
		foundPort := 0
		for _, port := range p.Ports {
			for _, host := range hosts {
				conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 1*time.Second)
				if err == nil {
					conn.Close()
					status = "running"
					foundPort = port
					break
				}
			}
			if status == "running" {
				break
			}
		}
		statuses = append(statuses, ServiceStatus{
			Name:   p.Name,
			Engine: p.Engine,
			Type:   p.Type,
			Status: status,
			Port:   foundPort,
		})
	}
	return statuses
}

// ── /proc readers ──

func (s *MetricsService) readCPUPercent() float64 {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 8 {
				var total, idle uint64
				for i := 1; i < len(fields); i++ {
					val, _ := strconv.ParseUint(fields[i], 10, 64)
					total += val
					if i == 4 { // idle
						idle = val
					}
				}

				if s.prevCPUTotal > 0 {
					totalDelta := float64(total - s.prevCPUTotal)
					idleDelta := float64(idle - s.prevCPUIdle)
					if totalDelta > 0 {
						cpuPercent := (1.0 - idleDelta/totalDelta) * 100
						s.prevCPUIdle = idle
						s.prevCPUTotal = total
						return cpuPercent
					}
				}
				s.prevCPUIdle = idle
				s.prevCPUTotal = total
			}
		}
	}
	return 0
}

func (s *MetricsService) readMemInfo(m *HostMetrics) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return
	}
	defer f.Close()

	var total, free, buffers, cached int64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		val, _ := strconv.ParseInt(parts[1], 10, 64)
		switch parts[0] {
		case "MemTotal:":
			total = val / 1024 // KB → MB
		case "MemFree:":
			free = val / 1024
		case "Buffers:":
			buffers = val / 1024
		case "Cached:":
			cached = val / 1024
		}
	}
	m.RAMTotalMB = total
	m.RAMUsedMB = total - free - buffers - cached
	if m.RAMUsedMB < 0 {
		m.RAMUsedMB = total - free
	}
	if total > 0 {
		m.RAMPercent = float64(m.RAMUsedMB) / float64(total) * 100
	}
}

func (s *MetricsService) readDiskUsage(m *HostMetrics) {
	// Read from /proc/mounts + statvfs-like via df approach
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return
	}
	defer f.Close()

	// We'll use a simple statfs approach via reading /proc/diskstats
	// For simplicity, read the root filesystem stats
	var stat StatFS
	if err := statfs("/", &stat); err == nil {
		totalBytes := stat.Blocks * uint64(stat.Bsize)
		freeBytes := stat.Bfree * uint64(stat.Bsize)
		m.DiskTotalGB = float64(totalBytes) / 1024 / 1024 / 1024
		m.DiskUsedGB = float64(totalBytes-freeBytes) / 1024 / 1024 / 1024
		if m.DiskTotalGB > 0 {
			m.DiskPercent = m.DiskUsedGB / m.DiskTotalGB * 100
		}
	}
}

func (s *MetricsService) readLoadAvg(m *HostMetrics) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return
	}
	parts := strings.Fields(string(data))
	if len(parts) >= 3 {
		m.LoadAvg1, _ = strconv.ParseFloat(parts[0], 64)
		m.LoadAvg5, _ = strconv.ParseFloat(parts[1], 64)
		m.LoadAvg15, _ = strconv.ParseFloat(parts[2], 64)
	}
}

func (s *MetricsService) readNetwork(m *HostMetrics) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return
	}
	defer f.Close()

	var totalRx, totalTx int64
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= 2 { // skip headers
			continue
		}
		line := scanner.Text()
		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) < 10 {
			continue
		}
		iface := strings.TrimSuffix(parts[0], ":")
		if iface == "lo" {
			continue // skip loopback
		}
		rx, _ := strconv.ParseInt(parts[1], 10, 64)
		tx, _ := strconv.ParseInt(parts[9], 10, 64)
		totalRx += rx
		totalTx += tx
	}
	m.NetworkRxMB = float64(totalRx) / 1024 / 1024
	m.NetworkTxMB = float64(totalTx) / 1024 / 1024
}

func (s *MetricsService) readUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "unknown"
	}
	parts := strings.Fields(string(data))
	if len(parts) < 1 {
		return "unknown"
	}
	secs, _ := strconv.ParseFloat(parts[0], 64)
	days := int(secs) / 86400
	hours := (int(secs) % 86400) / 3600
	mins := (int(secs) % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	return fmt.Sprintf("%dh %dm", hours, mins)
}

func (s *MetricsService) readProcessCount() int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			if _, err := strconv.Atoi(e.Name()); err == nil {
				count++
			}
		}
	}
	return count
}

func getGatewayIP() string {
	data, err := os.ReadFile("/proc/net/route")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[1] == "00000000" {
				gw := fields[2]
				if len(gw) == 8 {
					var a, b, c, d uint64
					fmt.Sscanf(gw[6:8], "%x", &a)
					fmt.Sscanf(gw[4:6], "%x", &b)
					fmt.Sscanf(gw[2:4], "%x", &c)
					fmt.Sscanf(gw[0:2], "%x", &d)
					return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
				}
			}
		}
	}
	return "172.17.0.1"
}
