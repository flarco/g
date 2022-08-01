package g

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/jaypipes/ghw"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
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
	respBytes, err := ioutil.ReadAll(resp.Body)
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

// GetMachineInfo obtains host information
func GetMachineInfo() {
	product, err := ghw.Product()
	if err != nil {
		fmt.Printf("Error getting product info: %v", err)
	}

	fmt.Printf("%v\n", product)
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
