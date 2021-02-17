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
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
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

// GetMachineProcStats returns the machine performance metrics
func GetMachineProcStats() map[string]interface{} {
	statsMap := map[string]interface{}{}

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

	cpuPct, _ := cpu.Percent(0, false)
	cpuTime, _ := cpu.Times(false)
	memRAM, _ := mem.VirtualMemory()

	if len(cpuPct) != 0 {
		statsMap["cpu_pct"] = cpuPct[0]
	}
	if len(cpuTime) != 0 {
		statsMap["cpu_time"] = cpuTime[0].Total()
	}
	statsMap["ram_pct"] = memRAM.UsedPercent
	statsMap["ram_rss"] = memRAM.Used
	statsMap["ram_total"] = memRAM.Total
	statsMap["tx_bytes"] = txBytes
	statsMap["rc_bytes"] = rcBytes

	return statsMap
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
