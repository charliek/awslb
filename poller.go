package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
)

// NewPoller creates the structure that will call the writter at at the
// polling interval or exit on a signal
func NewPoller(config Config, writer *writer) *Poller {
	return &Poller{
		config: config,
		writer: writer,
	}
}

// Poller is the object that will poll until exit signals are sent
type Poller struct {
	config Config
	writer *writer
}

// WriteFiles will write out files (if necessary) for all configured writers
func (c *Poller) WriteFiles() {
	c.writer.WriteTemplate()
}

func (c *Poller) loop() {
	log.Info("Polling AWS...")
	c.WriteFiles()
	interval := time.Duration(c.config.PollingSeconds)
	ticker := time.NewTicker(interval * time.Second).C
	done := make(chan os.Signal, 1)
	reload := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, os.Kill, syscall.SIGTERM)
	signal.Notify(reload, syscall.SIGHUP)

	running := true
	for running {
		select {
		case <-ticker:
			log.Info("Polling AWS...")
			c.WriteFiles()
		case <-reload:
			// TODO reload configuration
			log.Info("Reloading Configuration...Doesn't work yet...")
		case <-done:
			log.Info("Exiting...")
			running = false
		}
	}
}
