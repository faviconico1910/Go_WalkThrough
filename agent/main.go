package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
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

	sysChan := make(chan Payload)
	go getSystemInfo(sysChan, config)

	apiUrl := config.Agent.ApiURL
	processAndSend(sysChan, apiUrl, config)
}
