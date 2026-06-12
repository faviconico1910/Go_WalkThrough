package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
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
		CPU  bool `yaml:"cpu"`
		RAM  bool `yaml:"ram"`
		DISK bool `yaml: "disk"`
	} `yaml:"collectors"`
}

type Resource struct {
	Host       string `json:"host"`
	IPAddress  string `json:"ip_address"`
	OSPlatform string `json:"os_platform"`
	OSVersion  string `json:"os_version"`
	Uptime     int64  `json:"uptime_seconds"`
}

type Metric struct {
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Unit      string            `json:"unit"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

type Service struct {
	Name           string  `json:"name"`
	Status         string  `json:"status"`
	Port           int     `json:"port"`
	ResponseTimeMs float64 `json:"response_time_ms"`
	Timestamp      int64   `json:"timestamp"`
}

type Payload struct {
	Resource Resource  `json:"resource"`
	Metrics  []Metric  `json:"metrics"`
	Services []Service `json:"services"`
}

// build Otel payload
func buildOtelPayload(resource Resource, metrics []Metric, services []Service) Payload {
	return Payload{
		Resource: resource,
		Metrics:  metrics,
		Services: services,
	}
}

func collectResource() (Resource, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return Resource{}, err
	}

	hostInfo, err := host.Info()
	if err != nil {
		return Resource{}, err
	}

	ipList, err := getLocalIP()
	if err != nil {
		return Resource{}, err
	}

	return Resource{
		Host:       hostname,
		IPAddress:  ipList[0],
		OSPlatform: hostInfo.Platform,
		OSVersion:  hostInfo.PlatformVersion,
		Uptime:     int64(hostInfo.Uptime),
	}, nil

}

// hàm lấy địa chỉ IP của máy chủ
func getLocalIP() ([]string, error) {
	var ipList []string
	addresses, err := net.InterfaceAddrs() // Collect all system interfaces
	if err != nil {
		return nil, err
	}

	for _, addr := range addresses {
		// Check if the address belongs to an IP network
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			// Filter for valid IPv4 addresses
			if ipNet.IP.To4() != nil {
				ipList = append(ipList, ipNet.IP.String())
			}
		}
	}
	return ipList, nil
}

func getSystemInfo(ch chan<- Payload, config Config) {
	interval := time.Duration(config.Agent.Interval) * time.Second

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		metric := []Metric{}
		// get resource info
		resource, err := collectResource()
		if err != nil {
			fmt.Printf("[ERROR]: collecting resource failed: %v\n", err)
			continue
		}

		// Get CPU usage
		if config.Collectors.CPU {
			cpuPercent, err := cpu.Percent(time.Second, false)
			if err != nil {
				fmt.Printf("[ERROR]: collecting cpu usage failed: %v\n", err)
				continue
			}
			metric = append(metric, Metric{
				Name:      "system.cpu.utilization",
				Value:     cpuPercent[0],
				Unit:      "%",
				Timestamp: time.Now().Unix(),
				Tags: map[string]string{
					"cpu_core": "all",
					"state":    "user",
				},
			})
		}

		// Get RAM usage
		if config.Collectors.RAM {
			ramUsage, err := mem.VirtualMemory()
			if err != nil {
				fmt.Printf("[ERROR]: collecting ram usage failed: %v\n", err)
				continue
			}
			metric = append(metric, Metric{
				Name:      "system.memory.usage",
				Value:     ramUsage.UsedPercent,
				Unit:      "%",
				Timestamp: time.Now().Unix(),
				Tags: map[string]string{
					"state": "used",
				},
			})
		}

		// get disk utilization
		if config.Collectors.DISK {
			diskUsage, err := disk.Usage("C:\\")
			if err != nil {
				fmt.Printf("[ERROR]: collecting disk usage failed: %v\n", err)
				continue
			}
			metric = append(metric, Metric{
				Name:      "system.disk.utilization",
				Value:     diskUsage.UsedPercent,
				Unit:      "%",
				Timestamp: time.Now().Unix(),
				Tags: map[string]string{
					"mount_point": diskUsage.Path,
				},
			})
		}

		service := []Service{}
		payload := buildOtelPayload(resource, metric, service)

		ch <- payload
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

	sysChan := make(chan Payload)
	go getSystemInfo(sysChan, config)

	for payload := range sysChan {
		jsonData, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			fmt.Printf("[ERROR]: Error marshalling JSON: %v\n", err)
			continue
		}
		fmt.Println(string(jsonData))
		fmt.Print("-----------------------------------------------------")
	}

}
