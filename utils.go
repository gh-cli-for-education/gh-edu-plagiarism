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
	"time"

	"github.com/go-ping/ping"
	"github.com/spf13/viper"
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

func check() []error {
	fmt.Println("Checking everything is ok...")
  fmt.Println(basepath)
	dependencies := map[string]string{
		"fzf":    "You need to have fzf installed\nhttps://github.com/junegunn/fzf",
		"mossum": "You need to have mossum installed\nhttps://github.com/hjalti/mossum",
		"perl":   "You need to have perl installed",
		fmt.Sprintf("%s/moss", basepath): "You need to have a moss script in the root\nhttps://theory.stanford.edu/~aiken/moss/",
	}
	errorS := make([]error, 0, len(dependencies)+5)
	for d, e := range dependencies {
		if _, err := exec.LookPath(d); err != nil {
			errorS = append(errorS, errors.New(e))
		}
	}
	// Check python version 3
	if r, err := executeCmd(`python -c "print(__import__('sys').version_info[:1]==(3,))"`, false, nil); r != "True" {
		errorS = append(errorS, errors.New("Python version 3 is required\n"+err.Error()))
	}
	pinger, err := ping.NewPinger("moss.stanford.edu")
	if err != nil {
		log.Fatal(err)
	}
	pinger.Count = 2
	pinger.Timeout = time.Second * 2
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		errorS = append(errorS, err)
	}
	if pinger.Statistics().PacketsRecv == 0 {
		errorS = append(errorS, errors.New("couldn't conect to the server"))
	}
	if viper.GetString("defaultOrg") == "" {
		errorS = append(errorS, errors.New("please set an organization"))
	}
	if viper.GetString("assignment") == "" {
		errorS = append(errorS, errors.New("please set a current assignment"))
	}
	return errorS
}
