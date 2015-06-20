package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"text/template"

	log "github.com/Sirupsen/logrus"
)

type writer struct {
	config Config
	//lookup     *ConsulLookUp
	lastRunMd5 [16]byte
}

// newWriter create a writer that will look for changes and if needed
// write out a new file and reload the service.
func newWriter(c Config) *writer {
	// client := ...
	return &writer{
		config: c,
	}
}

type AwsLookup interface {
	Lookup() ([]*Instance, error)
}

func (w *writer) loadTemplate() (*template.Template, error) {
	log.Debugf("Loading template from %s", w.config.Source)
	funcMap := template.FuncMap{
		"lookupService": w.LookupService,
	}
	return template.New("conf").Funcs(funcMap).ParseFiles(w.config.Source)
}

func (w *writer) LookupService(name string) ([]*Instance, error) {
	log.Debugf("Looking up service %s", name)
	service, ok := w.config.Services[name]
	if !ok {
		log.Errorf("Service named %s was not found", name)
		return nil, fmt.Errorf("Service named %s was not found", name)
	}

	var lookup AwsLookup
	switch service.Type {
	case "elb":
		lookup = NewElbLookup(service.Elb)
	case "search":
		lookup = NewSearchLookup(service.Search)
	default:
		return nil, fmt.Errorf("Invalid type of %s found", service.Type)
	}
	return lookup.Lookup()
}

func (w *writer) reloadService() {
	if w.config.Command != "" {
		log.Infof("Reloading service with command %s", w.config.Command)
		success := executeTask(w.config.Command)
		if success {
			log.Info("Reload command finished successfully")
		} else {
			log.Warn("Reload command finished with an error")
		}
	}
}

func (w *writer) checkConfig([]byte) error {
	if w.config.Check != "" {
		// TODO need to implement configuration checking
		log.Infof("Checking config with command %s", w.config.Check)
	}
	return nil
}

func (w *writer) WriteTemplate() error {
	tmpl, err := w.loadTemplate()
	if err != nil {
		return err
	}
	path := w.config.Destination

	ctx := map[string][]string{}
	var buff bytes.Buffer
	err = tmpl.ExecuteTemplate(&buff, w.config.Source, ctx)
	if err != nil {
		log.Errorf("Error writing template to '%s': %v", path, err)
		return err
	}
	bits := buff.Bytes()
	checksum := md5.Sum(bits)
	if checksum != w.lastRunMd5 {
		w.lastRunMd5 = checksum
		err = w.checkConfig(bits)
		if err != nil {
			log.Errorf("Not writing file %s. Error with dynamic temlate output %v", path, err)
			return err
		}

		ioutil.WriteFile(path, bits, 0644)
		// TODO figure out how to chmod to proper user/group. Maybe shell out if needed.
		// os.Chown(path, t.Uid, t.Gid)
		// os.Chmod(path, w.config.FileMode)
		log.Infof("Wrote out file %s", path)
		w.reloadService()
	} else {
		log.Info("Not writing out file since there was no change")
	}
	return nil
}
