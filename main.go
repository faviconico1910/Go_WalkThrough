package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"gopkg.in/yaml.v3"
)

// create config struct
type Config struct {
	Agent struct {
		Name     string `yaml:"name"`
		Interval int    `yaml:"interval_seconds"`
	} `yaml:"agent"`

	Collectors struct {
		CPU bool `yaml:"cpu"`
		RAM bool `yaml:"ram"`
	} `yaml:"collectors"`
}

type MetricDataPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type Metric struct {
	Name       string            `json:"name"`
	Unit       string            `json:"unit"`
	DataPoints []MetricDataPoint `json:"data_points"`
}

// định nghĩa otel payload
type OtelPayload struct {
	Resource struct {
		ServiceName string `json:"service.name"`
	} `json:"resource"`

	Metrics []Metric `json:"metrics"`
}

type SystemInfo struct {
	Name     string  `json:"server_name"`
	CPU      float64 `json:"cpu_usage"`
	RAM      float64 `json:"ram_usage"`
	Time     string  `json:"timestamp"`
	ErrorMsg string  `json:"error_msg"`
}

// convert SystemInfo to OtelPayload

func buildOtelPayload(sysInfo SystemInfo) OtelPayload {
	var payload OtelPayload
	payload.Resource.ServiceName = sysInfo.Name

	payload.Metrics = []Metric{
		{
			Name: "cpu_usage",
			Unit: "%",
			DataPoints: []MetricDataPoint{
				{
					Value:     sysInfo.CPU,
					Timestamp: sysInfo.Time,
				},
			},
		},

		{
			Name: "ram_usage",
			Unit: "%",
			DataPoints: []MetricDataPoint{
				{
					Value:     sysInfo.RAM,
					Timestamp: sysInfo.Time,
				},
			},
		},
	}
	return payload
}

func getSystemInfo(ch chan<- SystemInfo, config Config) {

	interval := time.Duration(config.Agent.Interval) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		// Get CPU usage
		cpuPercent, err := cpu.Percent(time.Second, false)
		if err != nil {
			ch <- SystemInfo{
				Name:     config.Agent.Name,
				Time:     time.Now().Format(time.RFC3339),
				ErrorMsg: fmt.Sprintf("[ERROR]: Error getting CPU usage failed: %v", err),
			}
			continue
		}

		// Get RAM usage
		ramUsage, err := mem.VirtualMemory()
		if err != nil {
			ch <- SystemInfo{
				Name:     config.Agent.Name,
				Time:     time.Now().Format(time.RFC3339),
				ErrorMsg: fmt.Sprintf("[ERROR]: getting RAM usage failed: %v", err),
			}
			continue
		}

		sysInfo := SystemInfo{
			Name: config.Agent.Name,
			CPU:  cpuPercent[0],
			RAM:  ramUsage.UsedPercent,
			Time: time.Now().Format(time.RFC3339),
		}
		ch <- sysInfo // gửi dữ liệu vào channel
	}

}
func main() {

	// đọc yaml file
	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Printf("[ERROR]: Error reading config file: %v\n", err)
		return
	}

	var config Config

	// unmarshal yaml file
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Printf("[ERROR]: Error parsing config file: %v\n", err)
		return
	}

	sysChan := make(chan SystemInfo)
	go getSystemInfo(sysChan, config)

	for sysMetrics := range sysChan {
		if sysMetrics.ErrorMsg != "" {
			fmt.Println(sysMetrics.ErrorMsg)
			continue
		}

		payload := buildOtelPayload(sysMetrics)
		jsonBytes, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			fmt.Printf("[ERROR]: Error marshaling payload: %v\n", err)
			continue
		}

		fmt.Println(string(jsonBytes))

		// fmt.Printf(
		// 	"[INFO] System Info:\nServer Name: %s\nCPU Usage: %s\nRAM Usage: %s\nTimestamp: %s\n",
		// 	sysMetrics.Name,
		// 	sysMetrics.CPU,
		// 	sysMetrics.RAM,
		// 	sysMetrics.Time,
		// )
	}

}
