package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

const (
	PENDING  = "PENDING"
	ERROR    = "ERROR"
	COMPLETE = "COMPLETE"
)

type TaskStatus struct {
	Status string
}

func executeTask(cmdString string) bool {
	cmd := exec.Command("bash", "-c", cmdString)
	commandOut := make(chan string)
	status := &TaskStatus{PENDING}
	go runCommand(cmd, commandOut, status)
	for line := range commandOut {
		log.Info(line)
	}
	return status.Status == COMPLETE
}

func runCommand(cmd *exec.Cmd, commandOut chan string, status *TaskStatus) {
	defer close(commandOut)
	status.Status = PENDING
	outPipe, err := cmd.StdoutPipe()
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Warnf("Error reading command output %s", err)
		status.Status = ERROR
		return
	}
	cmd.Start()
	go readPipeOutput(outPipe, commandOut, "out > ")
	go readPipeOutput(errPipe, commandOut, "err > ")

	err = cmd.Wait()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				// The program has exited with an exit code != 0

				// This works on both Unix and Windows. Although package
				// syscall is generally platform dependent, WaitStatus is
				// defined for both Unix and Windows and in both cases has
				// an ExitStatus() method with the same signature.
				log.Errorf("Exit Status: %d", status.ExitStatus())
			} else {
				log.Errorf("Unknown error when running command %#v", err)
			}
		} else {
			log.Errorf("Unknown error when running command %#v", err)
		}
		status.Status = ERROR
	} else {
		status.Status = COMPLETE
	}
}

func readPipeOutput(pipe io.ReadCloser, commandOut chan string, prefix string) {
	stdout := bufio.NewReader(pipe)
	prefixLen := len(prefix)
	for {
		line, err := stdout.ReadString('\n')
		if err == nil || err == io.EOF {
			line = fmt.Sprintf("%s%s", prefix, strings.TrimSpace(line))
			if len(line) > prefixLen {
				commandOut <- line
			}
		}
		if err != nil {
			break
		}
	}
}
