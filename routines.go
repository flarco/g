package g

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/docker"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/spf13/cast"
)

var publicIPTimestamp time.Time

// PublicIP is the public IP
var PublicIP string

// UpdatePublicIP updates the public IP value
func UpdatePublicIP() error {
	if !publicIPTimestamp.IsZero() && time.Since(publicIPTimestamp).Seconds() < 60*60 {
		return nil
	}

	defer func() { publicIPTimestamp = time.Now() }()

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://ifconfig.me")
	if err != nil {
		return Error(err, "Could not Get IP from http://ifconfig.me")
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Error(err, "Could not read IP response from http://ifconfig.me")
	}
	PublicIP = string(respBytes)
	return nil
}

type ProcStats struct {
	CpuPct     float64
	CpuTime    float64
	RamPct     float64
	RamRss     uint64
	RamTotal   uint64
	DiskPct    float64
	DiskFree   uint64
	ReadBytes  uint64
	WriteBytes uint64
	TxBytes    uint64
	RcBytes    uint64
}

// GetMachineProcStats returns the machine performance metrics
func GetMachineProcStats() ProcStats {
	stats := ProcStats{}

	netCounters, _ := net.IOCounters(true)

	txBytes := uint64(0)
	rcBytes := uint64(0)
	for _, netCounter := range netCounters {
		if netCounter.Name == "lo" {
			continue
		}
		txBytes = rcBytes + netCounter.BytesSent
		rcBytes = rcBytes + netCounter.BytesRecv
	}
	stats.TxBytes = txBytes
	stats.RcBytes = rcBytes

	cpuPct, _ := cpu.Percent(0, false)
	cpuTime, _ := cpu.Times(false)
	memRAM, _ := mem.VirtualMemory()
	diskUsage, _ := disk.Usage("/")

	if len(cpuPct) != 0 {
		stats.CpuPct = cpuPct[0]
	}
	if len(cpuTime) != 0 {
		stats.CpuTime = cpuTime[0].Total()
	}

	if memRAM != nil {
		stats.RamPct = memRAM.UsedPercent
		stats.RamRss = memRAM.Used
		stats.RamTotal = memRAM.Total
	}

	if diskUsage != nil {
		stats.DiskPct = diskUsage.UsedPercent
		stats.DiskFree = diskUsage.Free
	}

	return stats
}

func GetContainerStats(containerID string) ProcStats {
	stats := ProcStats{}

	cpuStats, err := docker.CgroupCPUDocker(containerID)
	if err == nil {
		stats.CpuPct = cpuStats.Usage
		stats.CpuTime = cpuStats.Total()
	}

	memStats, err := docker.CgroupMemDocker(containerID)
	if err == nil {
		stats.RamPct = float64(memStats.MemUsageInBytes) / float64(memStats.MemLimitInBytes)
		stats.RamRss = memStats.RSS
		stats.RamTotal = memStats.MemLimitInBytes
	}

	return stats
}

func GetProcStats(pid int) ProcStats {
	stats := ProcStats{}
	proc, err := process.NewProcess(cast.ToInt32(pid))
	if err != nil {
		return stats
	}

	cpuPct, err := proc.CPUPercent()
	if err == nil {
		stats.CpuPct = cpuPct
	}

	cpuTime, err := proc.Times()
	if err == nil {
		stats.CpuTime = cpuTime.Total()
	}

	ramPct, err := proc.MemoryPercent()
	if err == nil {
		stats.RamPct = cast.ToFloat64(ramPct)
	}

	ramInfo, err := proc.MemoryInfo()
	if err == nil {
		stats.RamRss = ramInfo.RSS
	}

	// netInfo, err := proc.NetIOCounters(false)
	// if err == nil {
	// 	stats.TxBytes = cast.ToUint64(netInfo[0].BytesSent)
	// 	stats.RcBytes = cast.ToUint64(netInfo[0].BytesRecv)
	// }

	diskInfo, err := proc.IOCounters()
	if err == nil {
		stats.ReadBytes = diskInfo.ReadBytes
		stats.WriteBytes = diskInfo.WriteBytes
	}

	return stats
}

type (
	caller struct {
		Function string
		Path     string
	}

	// Routine is a go routine
	Routine struct {
		Number  int
		State   string
		Callers []caller
	}
)

// GetRunningGoRoutines returns the stack of all running goroutines
func GetRunningGoRoutines() (routines []Routine) {

	buf := make([]byte, 1<<16)
	l := runtime.Stack(buf, true)
	s := string(buf[0:l])
	for _, stackStr := range strings.Split(s, "\n\n") {
		r := Routine{}
		regex := *regexp.MustCompile(`goroutine (\d+) \[(\S+)\]`)
		res := regex.FindStringSubmatch(stackStr)
		if len(res) == 2+1 {
			r.Number = cast.ToInt(res[1])
			r.State = cast.ToString(res[2])
		}
		stackStrArr := strings.Split(stackStr, "\n")
		for i, line := range stackStrArr {
			if i == 0 {
				continue
			}
			if strings.HasPrefix(line, "\t") {
				c := caller{
					stackStrArr[i-1],
					strings.Split(strings.TrimSpace(line), " ")[0],
				}
				r.Callers = append(r.Callers, c)
			}
		}
		routines = append(routines, r)
	}

	return
}

// GetOwnContainerID returns the container ID of the current process
// Returns empty string if not running inside a container
func GetOwnContainerID() (string, error) {
	return GetOwnContainerIDWithContext(context.Background())
}

func GetOwnContainerIDWithContext(ctx context.Context) (string, error) {
	contents, err := ReadLines("/proc/self/cgroup")
	if err != nil {
		return "", err
	}

	for _, line := range contents {
		fields := strings.Split(line, ":")
		if len(fields) != 3 {
			continue
		}

		// Look for the container ID in the cgroup path
		parts := strings.Split(fields[2], "/")
		for _, part := range parts {
			// Docker container IDs are 64 characters long
			if len(part) == 64 {
				return part, nil
			}
			// Also check for docker prefix
			if strings.HasPrefix(part, "docker-") {
				return strings.TrimPrefix(part, "docker-"), nil
			}
		}
	}

	return "", nil
}

// ReadLines reads contents from a file and splits them by new lines.
// A convenience wrapper to ReadLinesOffsetN(filename, 0, -1).
func ReadLines(filename string) ([]string, error) {
	return ReadLinesOffsetN(filename, 0, -1)
}

// ReadLinesOffsetN reads contents from file and splits them by new line.
// The offset tells at which line number to start.
// The count determines the number of lines to read (starting from offset):
// n >= 0: at most n lines
// n < 0: whole file
func ReadLinesOffsetN(filename string, offset uint, n int) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string

	r := bufio.NewReader(f)
	for i := uint(0); i < uint(n)+offset || n < 0; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				ret = append(ret, strings.Trim(line, "\n"))
			}
			break
		}
		if i < offset {
			continue
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}

	return ret, nil
}
