package main

type Config struct {
	Agent struct {
		Name     string `yaml:"name"`
		Interval int    `yaml:"interval_seconds"`
	} `yaml:"agent"`

	Collectors struct {
		CPU      bool `yaml:"cpu"`
		RAM      bool `yaml:"ram"`
		DISK     bool `yaml:"disk"`
		Services bool `yaml:"services"`
	} `yaml:"collectors"`

	Services []ServiceConfig `yaml:"services"`
}

type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}
