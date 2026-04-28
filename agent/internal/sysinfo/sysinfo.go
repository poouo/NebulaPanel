package sysinfo

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Stats represents one snapshot of host stats.
type Stats struct {
	Token       string  `json:"token,omitempty"`
	Hostname    string  `json:"hostname"`
	Host        string  `json:"host"`
	CPU         float64 `json:"cpu"`
	CPUCores    int     `json:"cpu_cores"`
	CPUModel    string  `json:"cpu_model"`
	Mem         float64 `json:"mem"`
	MemTotal    int64   `json:"mem_total"`
	DiskTotal   int64   `json:"disk_total"`
	DiskUsed    int64   `json:"disk_used"`
	DiskUsage   float64 `json:"disk_usage"`
	NetIn       int64   `json:"net_in"`
	NetOut      int64   `json:"net_out"`
	NetInSpeed  int64   `json:"net_in_speed"`
	NetOutSpeed int64   `json:"net_out_speed"`
	Uptime      int     `json:"uptime"`
	Version     string  `json:"version"`
	XrayVersion string  `json:"xray_version,omitempty"`
	OS          string  `json:"os"`
	Kernel      string  `json:"kernel"`
	Arch        string  `json:"arch"`
	LoadAvg     string  `json:"load_avg"`
}

var (
	prevCPUTotal  uint64
	prevCPUIdle   uint64
	prevNetIn     int64
	prevNetOut    int64
	prevNetSample time.Time
	cachedCores   int
	cachedModel   string
	cachedKernel  string
	cachedOSPre   string
)

// Collect returns current snapshot. version is reported back to the panel.
func Collect(version string) Stats {
	s := Stats{
		Version: version,
		OS:      osPretty(),
		Arch:    runtime.GOARCH,
		Kernel:  kernelVersion(),
	}
	s.Hostname, _ = os.Hostname()
	s.Host = guessOutboundIP()
	s.Uptime = readUptime()
	s.MemTotal, s.Mem = readMemory()
	s.CPU = readCPU()
	s.CPUCores, s.CPUModel = cpuInfo()
	s.DiskTotal, s.DiskUsed, s.DiskUsage = diskUsage("/")
	s.NetIn, s.NetOut = readNet()
	s.NetInSpeed, s.NetOutSpeed = netSpeed(s.NetIn, s.NetOut)
	s.LoadAvg = loadAverage()
	return s
}

func osPretty() string {
	if cachedOSPre != "" {
		return cachedOSPre
	}
	if b, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				v := strings.TrimPrefix(line, "PRETTY_NAME=")
				cachedOSPre = strings.Trim(v, `"`)
				return cachedOSPre
			}
		}
	}
	cachedOSPre = runtime.GOOS
	return cachedOSPre
}

func kernelVersion() string {
	if cachedKernel != "" {
		return cachedKernel
	}
	var u syscall.Utsname
	if err := syscall.Uname(&u); err == nil {
		b := make([]byte, 0, len(u.Release))
		for _, c := range u.Release {
			if c == 0 {
				break
			}
			b = append(b, byte(c))
		}
		cachedKernel = string(b)
		return cachedKernel
	}
	if b, err := exec.Command("uname", "-r").Output(); err == nil {
		cachedKernel = strings.TrimSpace(string(b))
		return cachedKernel
	}
	return ""
}

func cpuInfo() (int, string) {
	if cachedCores > 0 {
		return cachedCores, cachedModel
	}
	cachedCores = runtime.NumCPU()
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					cachedModel = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}
	if cachedModel == "" {
		cachedModel = runtime.GOARCH
	}
	return cachedCores, cachedModel
}

func diskUsage(path string) (int64, int64, float64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, 0
	}
	total := int64(st.Blocks) * int64(st.Bsize)
	free := int64(st.Bavail) * int64(st.Bsize)
	used := total - free
	if total == 0 {
		return 0, 0, 0
	}
	pct := float64(used) / float64(total) * 100.0
	return total, used, pct
}

func loadAverage() string {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return ""
	}
	f := strings.Fields(string(b))
	if len(f) < 3 {
		return ""
	}
	return fmt.Sprintf("%s %s %s", f[0], f[1], f[2])
}

func netSpeed(curIn, curOut int64) (int64, int64) {
	now := time.Now()
	if prevNetSample.IsZero() {
		prevNetIn = curIn
		prevNetOut = curOut
		prevNetSample = now
		return 0, 0
	}
	dt := now.Sub(prevNetSample).Seconds()
	if dt <= 0 {
		return 0, 0
	}
	inSpd := int64(float64(curIn-prevNetIn) / dt)
	outSpd := int64(float64(curOut-prevNetOut) / dt)
	if inSpd < 0 {
		inSpd = 0
	}
	if outSpd < 0 {
		outSpd = 0
	}
	prevNetIn = curIn
	prevNetOut = curOut
	prevNetSample = now
	return inSpd, outSpd
}

func guessOutboundIP() string {
	conn, err := net.DialTimeout("udp", "1.1.1.1:80", 2*time.Second)
	if err == nil {
		defer conn.Close()
		if a, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			return a.IP.String()
		}
	}
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
