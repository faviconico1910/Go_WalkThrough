package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
	"gopkg.in/yaml.v3"
)

type program struct {
	config Config
}

func (p *program) Start(s service.Service) error {
	go p.runAgent()
	return nil
}

func (p *program) Stop(s service.Service) error {
	fmt.Println("[INFO]: Agent Background Service đang dừng...")
	return nil
}

func (p *program) runAgent() {
	fmt.Println("[INFO]: Agent Background Service đang vận hành...")

	sysChan := make(chan Payload)
	go getSystemInfo(sysChan, p.config)

	apiUrl := p.config.Agent.ApiURL
	processAndSend(sysChan, apiUrl, p.config)
}

func main() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("[ERROR]: Error getting executable path: %v\n", err)
		return
	}

	err = os.Chdir(filepath.Dir(exePath))
	if err != nil {
		fmt.Printf("[ERROR]: Error changing working directory: %v\n", err)
		return
	}

	yamlFile, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Printf("[ERROR]: Error reading config file: %v\n", err)
		return
	}

	var config Config
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		fmt.Printf("[ERROR]: Error parsing config file: %v\n", err)
		return
	}

	svcConfig := &service.Config{
		Name:        "MonitoringAgent",
		DisplayName: "Agent Background Service",
		Description: "A background service for monitoring system metrics and sending them to a specified API.",
	}

	prg := &program{config: config}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		fmt.Printf("[ERROR]: Error creating service: %v\n", err)
		return
	}

	if len(os.Args) > 1 {
		action := os.Args[1]
		err = service.Control(s, action)
		if err != nil {
			fmt.Printf("[ERROR]: Command Failed!: %v\n", err)
		}
		fmt.Printf("[INFO]: Service %s action executed successfully.\n", action)
		return
	}

	err = s.Run()
	if err != nil {
		fmt.Printf("[ERROR]: Service failed to run: %v\n", err)
	}
}
