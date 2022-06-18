package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
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
}

var (
	rootCmd = &cobra.Command{
		Use:   "gh edu plagiarism [-a] [-l <language>] [-t]",
		Short: "Detect plagiarism in students assigment",
		Long:  "gh-edu-plagiarism checks all the repositories from an assignment and compares it to detect plagiarism",
		RunE:  func(cmd *cobra.Command, args []string) error { return realMain() },
	}
	areTemplateF bool
	language     utils.Language
	anonymize    bool
)

// Clean up delete all the directories left by the last execution
// Is not done in the same execution because I don't know how much the user
// is going to need the files with commands like xdg-open
func cleanUp() error {
	tempDir := os.TempDir()
	if tempDir == "" {
		return errors.New("internal error: couldn't access temp dir")
	}
	pattern := tempDir + `/*gh-edu-plagiarism`
	filesString, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	for _, fString := range filesString {
		err = os.RemoveAll(fString)
		if err != nil {
			return err
		}
	}
	return nil
}

func getTemplate(reposC <-chan string, selectedTemplateC chan<- string) {
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
		fmt.Println(err)
	}
	selectedTemplateC <- result
}

type repoObj struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	dir  string
}

func clone(reposC <-chan repoObj, clonedReposC chan<- repoObj) {
	tmpDir, err := os.MkdirTemp("", "*_gh-edu-plagiarism")
	if err != nil {
		log.Fatal(err)
	}
	sem := make(chan empty, runtime.NumCPU())
	var wg sync.WaitGroup
	for repo := range reposC {
		sem <- empty{}
		wg.Add(1)
		go func(repo repoObj) {
			defer wg.Done()
			repoDir := filepath.Join(tmpDir, repo.Name)
			command := fmt.Sprintf("gh repo clone %s %s/", repo.Url, repoDir)
			_, err := utils.ExecuteCmd(command, false, nil)
			if err != nil {
				fmt.Println("error:", err)
			}
			repo.dir = repoDir
			clonedReposC <- repo
			<-sem
		}(repo)
	}
	wg.Wait()
	close(clonedReposC)
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
		os.Exit(1)
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
	go filter(sendToCloneC, selectTemplateC)
	go getTemplate(selectTemplateC, selectedTemplateC)
	clonedC := make(chan repoObj)
	go clone(sendToCloneC, clonedC)
	send(clonedC, selectedTemplateC)
	return nil
}

func check() []error {
	fmt.Println("Checking everything is ok...")
	mossPath := fmt.Sprintf("%s/moss", utils.Basepath)
	dependencies := map[string]string{
		"fzf":    "You need to have fzf installed\nhttps://github.com/junegunn/fzf",
		"mossum": "You need to have mossum installed\nhttps://github.com/hjalti/mossum",
		"perl":   "You need to have perl installed",
		mossPath: "You need to have a moss script in the root\nhttps://theory.stanford.edu/~aiken/moss/",
	}
  const posibleErr = 10
	errorS := make([]error, 0, posibleErr)
	for d, e := range dependencies {
		if _, err := exec.LookPath(d); err != nil {
			errorS = append(errorS, errors.New(e))
		}
	}
	// Check python version 3
	if r, err := utils.ExecuteCmd(`python -c "print(__import__('sys').version_info[:1]==(3,))"`, false, nil); r != "True" {
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
