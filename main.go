package main

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

type SystemInfo struct {
	Name     string `json:"server_name"`
	CPU      string `json:"cpu_usage"`
	RAM      string `json:"ram_usage"`
	Time     string `json:"timestamp"`
	ErrorMsg string `json:"error_msg"`
}

func getSystemInfo(ch chan<- SystemInfo) {

	for {
		// Get CPU usage
		cpuPercent, err := cpu.Percent(time.Second, false)
		if err != nil {
			fmt.Println("Error getting CPU usage:", err)
			return
		}

		// Get RAM usage
		ramUsage, err := mem.VirtualMemory()
		if err != nil {
			fmt.Println("Error getting RAM usage:", err)
			return
		}

		sysInfo := SystemInfo{
			Name: "Local Machine",
			CPU:  fmt.Sprintf("%.1f%%", cpuPercent[0]),
			RAM:  fmt.Sprintf("%.1f%%", ramUsage.UsedPercent),
			Time: time.Now().Format(time.RFC3339),
		}
		ch <- sysInfo // gửi dữ liệu vào channel
	}

}
func main() {

	sysChan := make(chan SystemInfo)
	go getSystemInfo(sysChan)

	for i := 1; i <= 5; i++ {
		sysMetrics := <-sysChan
		fmt.Printf("System Info lần %d: %+v\n", i, sysMetrics)
		time.Sleep(5 * time.Second)
	}
}
