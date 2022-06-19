package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	// _, b, _, _ = runtime.Caller(0)
	// Basepath   = filepath.Dir(b)

  dir, _ = os.Executable()
  Basepath = filepath.Dir(dir)
)

func AllReposQ(org string) string {
	return fmt.Sprintf(`
query($endCursor: String) {
  organization(login: "%s") {
    repositories(first: 100, after: $endCursor) {
      pageInfo {
        endCursor
        hasNextPage
      }
      edges {
        node  {
          name,
          url
        }
      }
    }
  }
}
`, org)
}

func ExecuteCmd(command string, showStderr bool, stdInFunc func(in io.Writer)) (string, error) {
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
		return "", errors.New(bError.String() + err.Error()) // TODO simplify to fmt.Errorf?
	}
	return strings.TrimRight(string(result), "\n"), nil
}

func ExecuteQuery(query string, options ...string) string {
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

func OpenFile(file string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", file).Run()
	case "windows":
		err = exec.Command("start", file).Run()
	case "darwin":
		err = exec.Command("open", file).Run()
	default:
		fmt.Println("Open the file", file)
	}
	return err
}

var LangOptions = [...]string{"c", "cc", "java", "ml", "pascal", "ada", "lisp", "scheme", "haskell", "fortran", "ascii", "vhdl", "perl", "matlab", "python", "mips", "prolog", "spice", "vb", "csharp", "modula2", "a8086", "javascript", "plsql", "verilog"}

type Language string

func (l Language) String() string {
	return string(l)
}

func (l *Language) Set(v string) error {
	contain := func(options []string, value string) bool {
		for _, o := range options {
			if value == o {
				return true
			}
		}
		return false
	}
	if contain(LangOptions[:], v) {
		*l = Language(v)
		return nil
	}
	return fmt.Errorf("must be one of: %v", LangOptions)
}

func (e Language) Type() string {
	return "https://github.com/gh-cli-for-education/gh-edu-plagiarism#compatible-languages"
}
