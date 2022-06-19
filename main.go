package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gh-cli-for-education/gh-edu-plagiarism/pkg/utils"
	"github.com/go-ping/ping"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type empty = struct{}

func init() {
	viper.SetConfigFile("../gh-edu/config.json")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error with configuration file: " + err.Error())
	}
	rootCmd.Flags().BoolVarP(&areTemplateF, "template", "t", false, "Indicate if there is a tutor template")
	rootCmd.Flags().VarP(&language, "language", "l", "Select the language")
	rootCmd.Flags().BoolVarP(&anonymize, "anonymize", "a", false, "Indicate if you want to randomize the names")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "No INFO in the output only the result")
}

var (
	rootCmd = &cobra.Command{
		Use:   "gh edu plagiarism [-a] [-q] [-l [<language>]] [-t]",
		Short: "Detect plagiarism in students assigment",
		Long:  "gh-edu-plagiarism checks all the repositories from an assignment and compares it to detect plagiarism",
		RunE:  func(cmd *cobra.Command, args []string) error { return realMain() },
	}
	areTemplateF bool
	language     langType
	anonymize    bool
	quiet        bool
)

// Clean up delete all the directories left by the last execution
// Is not done in the same execution because I don't know how much the user
// is going to need the files with commands like xdg-open
func cleanUp() error {
	tempDir := os.TempDir()
	if tempDir == "" {
		return fmt.Errorf("internal error: cleanUp: couldn't access temp dir")
	}
	pattern := tempDir + `/*gh-edu-plagiarism`
	filesString, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("internal error: cleanUp: %w", err)
	}
	for _, fString := range filesString {
		err = os.RemoveAll(fString)
		if err != nil {
			fmt.Println("internal warning: cleanUp: couldn't delete: ", fString)
		}
	}
	return nil
}

func selectTemplate(reposC <-chan string, selectedTemplateC chan<- string, errC chan<- error) {
	if reposC == nil || selectedTemplateC == nil {
		return
	}
	stdInFunc := func(in io.Writer) {
		for repo := range reposC {
			io.WriteString(in, repo+"\n")
		}
	}
	result, err := utils.ExecuteCmd("fzf", true, stdInFunc)
	if err != nil {
		errC <- fmt.Errorf("selectTemplate: %w", err)
		return
	}
	selectedTemplateC <- result
}

type repoObj struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	dir  string
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func realMain() error {
	errS := check()
	if len(errS) > 0 {
		for _, err := range errS {
			fmt.Println(err)
		}
		return fmt.Errorf("solve this/these problem(s) and try again")
	}
	err := cleanUp()
	if err != nil {
		return err
	}
	sendToCloneC := make(chan repoObj)
	selectTemplateC, selectedTemplateC := func() (chan string, chan string) {
		if areTemplateF {
			return make(chan string), make(chan string)
		}
		return nil, nil
	}()
	errC := make(chan error)

	go filter(sendToCloneC, selectTemplateC, errC)
	go selectTemplate(selectTemplateC, selectedTemplateC, errC)
	clonedC := make(chan repoObj)
	go clone(sendToCloneC, clonedC, errC)
	go send(clonedC, selectedTemplateC, errC) // send also close errC
	return <-errC
}

func check() []error {
	utils.Println(quiet, "Checking everything is ok...")
	mossPath := fmt.Sprintf("%s/moss", utils.Basepath)
	dependencies := map[string]string{
		"fzf":    "You need to have fzf installed\nhttps://github.com/junegunn/fzf",
		"mossum": "You need to have mossum installed\nhttps://github.com/hjalti/mossum",
		"perl":   "You need to have perl installed",
		mossPath: "You need to have a moss script in the root\nhttps://theory.stanford.edu/~aiken/moss/\nRoot: " + utils.Basepath,
	}
	const posibleErr = 10
	errorS := make([]error, 0, posibleErr)
	for d, e := range dependencies {
		if _, err := exec.LookPath(d); err != nil {
			errorS = append(errorS, fmt.Errorf(e))
		}
	}
	// Check python version 3
	if r, err := utils.ExecuteCmd(`python -c "print(__import__('sys').version_info[:1]==(3,))"`, false, nil); r != "True" {
		errorS = append(errorS, fmt.Errorf("Python version 3 is required\n"+err.Error()))
	}
	pinger, err := ping.NewPinger("moss.stanford.edu")
	if err != nil {
		errorS = append(errorS, fmt.Errorf("internal error: setting up ping: %w", err))
	}
	pinger.Count = 2
	pinger.Timeout = time.Second * 2
	err = pinger.Run() // Blocks until finished.
	if err != nil {
		errorS = append(errorS, err)
	}
	if pinger.Statistics().PacketsRecv == 0 {
		errorS = append(errorS, fmt.Errorf("couldn't connect to the server"))
	}
	if viper.GetString("defaultOrg") == "" {
		errorS = append(errorS, fmt.Errorf("please set an organization"))
	}
	if viper.GetString("assignment") == "" {
		errorS = append(errorS, fmt.Errorf("please set a current assignment"))
	}
	return errorS
}
