package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
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

// tạo 1 queue mới để lưu trữ payload khi offline
func NewMemoryQueue(capacity int) *MemoryQueue {
	return &MemoryQueue{
		buffer:   make([]Payload, 0, capacity),
		capacity: capacity,
	}
}

// queue và enqueue
func (q *MemoryQueue) Push(payload Payload) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.capacity <= 0 {
		q.buffer = append(q.buffer, payload)
		return
	}

	if len(q.buffer) >= q.capacity {
		q.buffer = q.buffer[1:]
	}
	q.buffer = append(q.buffer, payload)
}

func (q *MemoryQueue) Pop() (Payload, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.buffer) == 0 {
		return Payload{}, false
	}
	payload := q.buffer[0]
	q.buffer[0] = Payload{}
	q.buffer = q.buffer[1:]
	return payload, true
}

// tính độ dài buffer
func (q *MemoryQueue) Length() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.buffer)
}

// hàm gửi dữ liệu lên API Hub
func sendPayload(payload Payload, apiUrl string) (bool, error) {

	data, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	client := http.Client{Timeout: 5 * time.Second}

	resp, err := client.Post(apiUrl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return true, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		fmt.Printf("[ERROR] Error Code: (%d)", resp.StatusCode)
		return false, nil
	}
	// 5xx mới là lỗi server
	if resp.StatusCode >= 500 {
		return true, fmt.Errorf("[ERROR] Server error: received status code %d", resp.StatusCode)
	}
	return false, nil
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
		OSPlatform: hostInfo.Platform,
		OSVersion:  hostInfo.PlatformVersion,
		Uptime:     int64(hostInfo.Uptime),
		IPAddress:  firstLocalIP(ipList),
	}, nil
}

func firstLocalIP(ipList []string) string {
	if len(ipList) == 0 {
		return ""
	}
	return ipList[0]
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
		cpuPercent, err := cpu.Percent(0, false)
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

func processAndSend(ch <-chan Payload, apiUrl string, config Config) {
	queue := NewMemoryQueue(config.Buffer.MaxCapacity)

	for payload := range ch {
		jsonData, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			fmt.Printf("[ERROR]: Error marshalling JSON: %v\n", err)
			continue
		}
		fmt.Println(string(jsonData))
		fmt.Println("-----------------------------------------------------")
		if queue.Length() > 0 {
			isOnline := true
			for queue.Length() > 0 {
				oldPayload, ok := queue.Pop()
				if !ok {
					break
				}

				retry, err := sendPayload(oldPayload, apiUrl)
				if retry {
					queue.mu.Lock()
					queue.buffer = append([]Payload{oldPayload}, queue.buffer...)
					queue.mu.Unlock()
					fmt.Printf("[ERROR]: Failed to send old payload: %v. Will retry later.\n", err)
					isOnline = false
					break
				}
				fmt.Printf("[FLUSH] Successfully send old payload. Còn %d bản ghi\n", queue.Length())
				time.Sleep(time.Duration(config.Buffer.FlushInterval) * time.Millisecond)
			}

			if !isOnline {
				queue.Push(payload)
				fmt.Printf("[INFO]: RAM đang có %d bản ghi, sẽ gửi lại sau\n", queue.Length())
				continue
			}

			fmt.Println("[INFO]: RAM is empty now!")
		}

		retry, err := sendPayload(payload, apiUrl)
		if retry {
			queue.Push(payload)
			fmt.Printf("[ERROR]: Failed to send payload: %v. Will retry later.\n", err)
			fmt.Printf("[INFO]: Có %d bản ghi trong RAM\n", queue.Length())
		} else if err == nil {
			fmt.Println("[INFO]: Successfuly send payload to API Hub!")
		}

	}
}

func getSystemInfo(ch chan<- Payload, config Config) {
	interval := time.Duration(config.Agent.Interval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	_, _ = cpu.Percent(0, false)
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
