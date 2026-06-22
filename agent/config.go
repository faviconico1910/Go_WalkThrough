package main

type Config struct {
	Agent struct {
		Name     string `yaml:"name"`
		Interval int    `yaml:"interval_seconds"`
		ApiURL   string `yaml:"api_url"`
	} `yaml:"agent"`

	Collectors struct {
		CPU      bool `yaml:"cpu"`
		RAM      bool `yaml:"ram"`
		DISK     bool `yaml:"disk"`
		Services bool `yaml:"services"`
		Network  bool `yaml:"network"`
	} `yaml:"collectors"`

	Services []ServiceConfig `yaml:"services"`

	Buffer struct {
		MaxCapacity   int `yaml:"max_capacity"`
		FlushInterval int `yaml:"flush_interval_ms"`
	} `yaml:"buffer"`
}

type ServiceConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
}
