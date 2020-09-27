package gutil

import (
	"fmt"
	"github.com/jaypipes/ghw"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"io/ioutil"
	"net/http"
	"time"
)

var publicIPTimestamp time.Time

// PublicIP is the public IP
var PublicIP string

// UpdatePublicIP updates the public IP value
func UpdatePublicIP() {
	if !publicIPTimestamp.IsZero() && time.Since(publicIPTimestamp).Seconds() < 60*60 {
		return
	}

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://ifconfig.me")
	if err != nil {
		LogError(err, "Could not Get IP from http://ifconfig.me")
		return
	}
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		LogError(err, "Could not read IP response from http://ifconfig.me")
		return
	}
	PublicIP = string(respBytes)
	publicIPTimestamp = time.Now()
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
