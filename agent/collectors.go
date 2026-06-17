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
	"sync"
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

	if len(q.buffer) >= q.capacity {
		q.buffer[0] = Payload{}
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
func sendPayload(payload Payload, apiUrl string) error {

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling payload to JSON: %w", err)
	}

	client := http.Client{Timeout: 5 * time.Second}

	resp, err := client.Post(apiUrl, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}
	return nil
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

func processAndSend(ch <-chan Payload, apiUrl string) {
	// Khởi tạo hàng đợi RAM 300 bản ghi
	queue := NewMemoryQueue(300)
	isOffline := false

	var stateMu sync.Mutex

	for payload := range ch {
		jsonData, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			fmt.Printf("[ERROR]: Error marshalling JSON: %v\n", err)
			continue
		}
		fmt.Println(string(jsonData))
		fmt.Println("-----------------------------------------------------")

		stateMu.Lock()
		// offline
		if isOffline {
			queue.Push(payload)
			fmt.Printf("[WARN]: Đang offline. Ghi vào RAM Cache. Hiện có %d bản ghi trong RAM\n", queue.Length())
			stateMu.Unlock()
			fmt.Println("-----------------------------------------------------")
			continue
		}
		stateMu.Unlock()
		// online
		err = sendPayload(payload, apiUrl)
		if err != nil {
			// Mất mạng -> Đẩy vào Ring Buffer
			fmt.Printf("[WARN]: Kết nối thất bại (%v). Ghi vào RAM Cache.\n", err)
			queue.Push(payload)

			stateMu.Lock()
			isOffline = true
			stateMu.Unlock()

			// chạy worker routine
			go func() {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()

				for range ticker.C {
					if queue.Length() == 0 {
						stateMu.Lock()
						isOffline = false // trả lại trạng thái online sau khi đã gửi hết cache
						stateMu.Unlock()
						fmt.Println("[INFO]: Đã gửi hết cache. Trạng thái online trở lại.")
						fmt.Println("-----------------------------------------------------")
						return
					}

					testPayload, ok := queue.Pop()
					if !ok {
						fmt.Println("[INFO]: Không còn bản ghi trong RAM Cache.")
						continue
					}

					err := sendPayload(testPayload, apiUrl)
					if err != nil {
						queue.mu.Lock()
						queue.buffer = append([]Payload{testPayload}, queue.buffer...)
						queue.mu.Unlock()
						fmt.Println("[RETRY]: Thử kết nối lại thất bại")
						continue
					}
					fmt.Println("[SUCCESS]: Kết nối mạng đã phục hồi! Bắt đầu enqueue")
					for queue.Length() > 0 {
						payload, ok := queue.Pop()
						if !ok {
							break
						}

						_ = sendPayload(payload, apiUrl)
						fmt.Printf("[INFO]: Đã gửi 1 bản ghi từ RAM Cache. Còn lại %d bản ghi trong RAM\n", queue.Length())
						time.Sleep(1500 * time.Millisecond)
					}
					stateMu.Lock()
					isOffline = false
					stateMu.Unlock()
					return
				}
			}()
		} else {
			fmt.Println("[INFO]: Gửi dữ liệu thành công.")
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
