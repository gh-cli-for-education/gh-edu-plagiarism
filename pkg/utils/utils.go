package utils

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	usr, _ = user.Current()
	home   = usr.HomeDir
  ConfigPath = filepath.Join(home, ".config", "gh-edu", "data.json")

	dir, _   = os.Executable()
	Basepath = filepath.Dir(dir)
)

func AllReposQ(org string) string {
	return fmt.Sprintf(`
query($endCursor: String) {
  organization(login: "%s") {
    membersWithRole {
      totalCount
    },
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

func Println(silent bool, a ...any) (int, error) {
	if !silent {
		return fmt.Println(a...)
	}
	return 0, nil
}

func Min(a, b int) int {
	result := math.Min(float64(a), float64(b))
	return int(result)
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
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return "", fmt.Errorf("ExecuteCmd: stdin: %w", err)
		}
		go func() {
			stdInFunc(stdin)
			stdin.Close()
		}()
	}
	result, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf(bError.String() + err.Error())
	}
	return strings.TrimRight(string(result), "\n"), nil
}

// FzfCmd returns a customized fzf command
func FzfCmd(prompt string) string {
	if prompt != "" {
		return fmt.Sprintf("fzf --prompt='%s>' --layout=reverse --border", prompt)
	}
	return "fzf --layout=reverse --border"
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
		fmt.Println("Open the next file", file)
	}
	return err
}
