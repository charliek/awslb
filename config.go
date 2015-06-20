package main

import (
	"io/ioutil"

	log "github.com/Sirupsen/logrus"

	"gopkg.in/yaml.v2"
)

// Config is the structure that holds the program configuration.
// This mirrors the yaml file since it will be automatically bound
// to it.
type Config struct {
	PollingSeconds int `yaml:"polling_seconds"`
	Source         string
	Destination    string
	Command        string
	Check          string
	Services       map[string]ServiceConfig
}

// ServiceConfig represents the services configured in the yaml file.
type ServiceConfig struct {
	Type   string
	Elb    ElbConfig
	Search SearchConfig
}

// SearchConfig represents the configuration need to the search type
type SearchConfig struct {
	Region  string
	Filters []string
}

// ElbConfig represents the configuration needed for the "elb" type
type ElbConfig struct {
	CheckHealth bool `yaml:"check_health"`
	Name        string
	Region      string
}

func loadConfigFile(path string) Config {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("%v\n", err)
		log.Fatalf("Error reading configuration file %s", path)
	}
	config := Config{}
	if err := yaml.Unmarshal([]byte(data), &config); err != nil {
		log.Fatalf("Error parsing yaml file: %v", err)
	}
	return config
}
