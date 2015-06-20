package main

import (
	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	debug      = kingpin.Flag("debug", "Enable debug mode.").Bool()
	configFile = kingpin.Arg("config", "Location of configuration file.").Required().String()
)

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	log.Infof("Configuration loading from: %s", *configFile)
	config := loadConfigFile(*configFile)

	w := newWriter(config)

	configger := NewPoller(config, w)
	configger.loop()
}
