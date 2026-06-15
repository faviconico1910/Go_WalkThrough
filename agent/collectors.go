package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

func buildOtelPayload(resource Resource, metrics []Metric, services []Service) Payload {
	return Payload{
		Resource: resource,
		Metrics:  metrics,
		Services: services,
	}
}

// func checkWindowsService(svc ServiceConfig) Service {
// 	start := time.Now()

// 	cmd := exec.Command("sc", "query", svc.Name)
// 	output, err := cmd.Output()

// 	responseTimeMs := float64(time.Since(start).Microseconds()) / 1000.0
// 	status := "down"

// 	if err == nil {
// 		out := string(output)
// 		if strings.Contains(out, "RUNNING") {
// 			status = "up"
// 		}
// 	}

// 	return Service{
// 		Name:           svc.Name,
// 		Status:         status,
// 		Port:           svc.Port,
// 		ResponseTimeMs: responseTimeMs,
// 		Timestamp:      time.Now().Unix(),
// 	}
// }

func checkServiceStatus(svc ServiceConfig) Service {
	start := time.Now()
	status := "down"
	var cmd *exec.Cmd

	// nhận diện os
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("sc", "query", svc.Name)
	case "linux":
		cmd = exec.Command("systemctl", "is-active", svc.Name)
	default:
		fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
		return Service{
			Name:           svc.Name,
			Status:         "unknown",
			Port:           svc.Port,
			ResponseTimeMs: 0,
			Timestamp:      time.Now().Unix(),
		}
	}

	// execute
	output, err := cmd.Output()
	responseTimeMs := float64(time.Since(start).Microseconds()) / 1000.0

	outStr := strings.ToUpper(string(output))

	if runtime.GOOS == "windows" {
		if err == nil && strings.Contains(outStr, "RUNNING") {
			status = "up"
		}
	} else if runtime.GOOS == "linux" {
		if strings.Contains(outStr, "ACTIVE") && !strings.Contains(outStr, "INACTIVE") {
			status = "up"
		}
	}

	return Service{
		Name:           svc.Name,
		Status:         status,
		Port:           svc.Port,
		ResponseTimeMs: responseTimeMs,
		Timestamp:      time.Now().Unix(),
	}
}

func collectServices(services []ServiceConfig) []Service {
	svcList := []Service{}
	for _, svc := range services {
		svcList = append(svcList, checkServiceStatus(svc))
	}
	return svcList
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

func getLocalIP() ([]string, error) {
	var ipList []string
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addresses {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ipList = append(ipList, ipNet.IP.String())
			}
		}
	}
	return ipList, nil
}

func collectSystemMetrics(config Config) ([]Metric, error) {
	metrics := []Metric{}

	// lấy cpu usage
	if config.Collectors.CPU {
		cpuPercent, err := cpu.Percent(time.Second, false)
		if err != nil {
			return nil, fmt.Errorf("collecting cpu usage failed: %w", err)
		}
		metrics = append(metrics, Metric{
			Name:      "system.cpu.utilization",
			Value:     cpuPercent[0],
			Unit:      "%",
			Timestamp: time.Now().Unix(),
			Tags: map[string]string{
				"cpu_core": "all",
				"state":    "total",
			},
		})
	}

	// lấy ram usage
	if config.Collectors.RAM {
		ramUsage, err := mem.VirtualMemory()
		if err != nil {
			return nil, fmt.Errorf("collecting ram usage failed: %w", err)
		}
		UsedPercent := float64(ramUsage.Total-ramUsage.Available) / float64(ramUsage.Total) * 100.0
		metrics = append(metrics, Metric{
			Name:      "system.memory.usage",
			Value:     UsedPercent,
			Unit:      "%",
			Timestamp: time.Now().Unix(),
			Tags: map[string]string{
				"state": "used",
			},
		})
	}

	// lấy disk usage
	if config.Collectors.DISK {
		// mount point logic
		partitions, err := disk.Partitions(false)
		if err != nil {
			return nil, fmt.Errorf("getting disk partitions failed: %w", err)
		}

		for _, p := range partitions {
			diskUsage, err := disk.Usage(p.Mountpoint)
			if err != nil {
				fmt.Printf("[ERROR]: collecting disk usage for %s failed: %v\n", p.Mountpoint, err)
				continue
			}
			metrics = append(metrics, Metric{
				Name:      "system.disk.utilization",
				Value:     diskUsage.UsedPercent,
				Unit:      "%",
				Timestamp: time.Now().Unix(),
				Tags: map[string]string{
					"mount_point": diskUsage.Path,
				},
			})

		}

	}

	return metrics, nil
}

func getSystemInfo(ch chan<- Payload, config Config) {
	interval := time.Duration(config.Agent.Interval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		resource, err := collectResource()
		if err != nil {
			fmt.Printf("[ERROR]: collecting resource failed: %v\n", err)
			continue
		}

		metrics, err := collectSystemMetrics(config)
		if err != nil {
			fmt.Printf("[ERROR]: %v\n", err)
			continue
		}

		services := collectServices(config.Services)
		ch <- buildOtelPayload(resource, metrics, services)
	}
}
