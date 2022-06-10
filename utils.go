package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/go-ping/ping"
)

func executeCmd(command string, showStderr bool, stdInFunc func(in io.Writer)) (string, error) {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	cmd := exec.Command(shell, "-c", command)
	var bError bytes.Buffer
	if showStderr {
		cmd.Stderr = os.Stderr
	} else {
		cmd.Stderr = &bError
	}
	if stdInFunc != nil {
		stdin, _ := cmd.StdinPipe()
		go func() {
			stdInFunc(stdin)
			stdin.Close()
		}()
	}
	result, err := cmd.Output()
	if err != nil {
		return "", errors.New(bError.String() + err.Error())
	}
	fmt.Println("debug:", cmd.String())
	return strings.TrimRight(string(result), "\n"), nil
}

func executeQuery(query string, options ...string) string {
	commandString := []string{"api", "graphql", "--paginate"}
	if len(options) > 0 {
		commandString = append(commandString, options...)
	}
	commandString = append(commandString, "-f", fmt.Sprintf("query=%s", query))
	command := exec.Command("gh", commandString...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	result, err := command.Output()
	if err != nil {
		fmt.Println("Internal error:", err, ": ", stderr.String())
		fmt.Println(command.String())
		os.Exit(1)
	}
	return string(result)
}

func check() error {
	// registry := map[string]string{
 //    "fzf", "pepe"
 //  }
	// Check fzf is installed
	if _, err := exec.LookPath("fzf"); err != nil {
		return errors.New("You need to have fzf installed") // TODO delete this
	}
	// Check mossum is installed
	if _, err := exec.LookPath("mossum"); err != nil {
		return errors.New("You need to have mossum installed\nhttps://github.com/hjalti/mossum")
	}
	// Check moss script is present
	if _, err := exec.LookPath("./moss"); err != nil {
		return errors.New("You need to have a moss script in the root\nhttps://theory.stanford.edu/~aiken/moss/")
	}
	// Check perl is installed
	if _, err := exec.LookPath("perl"); err != nil {
		return errors.New("You need to have perl installed")
	}
	// Check python version 3
	if r, err := executeCmd(`python -c "print(__import__('sys').version_info[:1]==(3,))"`, false, nil); r != "True" {
		return fmt.Errorf("Python version 3 is required\n" + err.Error())
	}
	pinger, err := ping.NewPinger("moss.stanford.edu")
	if err != nil {
		return err
	}
	pinger.Count = 3
	log.Println("Checking server is up...")
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		return err
	}
	return nil
}
