package main

import "sync"

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

type MemoryQueue struct {
	mu       sync.Mutex
	capacity int
	buffer   []Payload
}
