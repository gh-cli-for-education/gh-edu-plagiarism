package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func executeCmd(command string, silent bool, stdInFunc func(in io.WriteCloser)) (string, error) {
	shell := os.Getenv("SHELL")
	if len(shell) == 0 {
		shell = "sh"
	}
	cmd := exec.Command(shell, "-c", command)
	if !silent {
		cmd.Stdout = os.Stdout
	}
	var errBuffer bytes.Buffer
	cmd.Stderr = &errBuffer
	if stdInFunc != nil {
		stdin, _ := cmd.StdinPipe()
		go func() {
			stdInFunc(stdin)
			stdin.Close()
		}()
	}
	result, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(errBuffer.String() + "\n" + err.Error())
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

// IndexFunc returns the first index i satisfying f(s[i]),
// or -1 if none do.
// func IndexFunc[E any](s []E, f func(E) bool) int {
// 	for i, v := range s {
// 		if f(v) {
// 			return i
// 		}
// 	}
// 	return -1
// }
