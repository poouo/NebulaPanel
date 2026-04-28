package sysinfo

import (
	"bufio"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Stats represents one snapshot of host stats.
type Stats struct {
	Hostname string  `json:"hostname"`
	Host     string  `json:"host"`
	CPU      float64 `json:"cpu"`
	Mem      float64 `json:"mem"`
	MemTotal int64   `json:"mem_total"`
	NetIn    int64   `json:"net_in"`
	NetOut   int64   `json:"net_out"`
	Uptime   int     `json:"uptime"`
	Version  string  `json:"version"`
	OS       string  `json:"os"`
	Arch     string  `json:"arch"`
}

var (
	prevCPUTotal uint64
	prevCPUIdle  uint64
)

// Collect returns current snapshot. version is reported back to the panel.
func Collect(version string) Stats {
	s := Stats{
		Version: version,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
	s.Hostname, _ = os.Hostname()
	s.Host = guessOutboundIP()
	s.Uptime = readUptime()
	s.MemTotal, s.Mem = readMemory()
	s.CPU = readCPU()
	s.NetIn, s.NetOut = readNet()
	return s
}

func guessOutboundIP() string {
	conn, err := net.DialTimeout("udp", "1.1.1.1:80", 2*time.Second)
	if err == nil {
		defer conn.Close()
		if a, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			return a.IP.String()
		}
	}
	// fallback: first non-loopback IPv4
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}

func readUptime() int {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	parts := strings.Fields(string(b))
	if len(parts) == 0 {
		return 0
	}
	f, _ := strconv.ParseFloat(parts[0], 64)
	return int(f)
}

func readMemory() (int64, float64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	var total, avail int64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "MemTotal:":
			total, _ = strconv.ParseInt(fields[1], 10, 64)
		case "MemAvailable:":
			avail, _ = strconv.ParseInt(fields[1], 10, 64)
		}
	}
	if total == 0 {
		return 0, 0
	}
	used := total - avail
	pct := float64(used) / float64(total) * 100.0
	return total * 1024, pct
}

func readCPU() float64 {
	b, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0
		}
		var total uint64
		var idle uint64
		for i, v := range fields[1:] {
			n, _ := strconv.ParseUint(v, 10, 64)
			total += n
			if i == 3 {
				idle = n
			}
		}
		dTotal := total - prevCPUTotal
		dIdle := idle - prevCPUIdle
		prevCPUTotal = total
		prevCPUIdle = idle
		if dTotal == 0 {
			return 0
		}
		return float64(dTotal-dIdle) / float64(dTotal) * 100.0
	}
	return 0
}

func readNet() (int64, int64) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	var in, out int64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		iface := strings.TrimSpace(line[:colon])
		if iface == "lo" || strings.HasPrefix(iface, "docker") || strings.HasPrefix(iface, "veth") || strings.HasPrefix(iface, "br-") {
			continue
		}
		fields := strings.Fields(line[colon+1:])
		if len(fields) < 16 {
			continue
		}
		ri, _ := strconv.ParseInt(fields[0], 10, 64)
		ti, _ := strconv.ParseInt(fields[8], 10, 64)
		in += ri
		out += ti
	}
	return in, out
}
