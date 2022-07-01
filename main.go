package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	// "time"

	"github.com/gh-cli-for-education/gh-edu-plagiarism/pkg/utils"
	// "github.com/go-ping/ping"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type empty = struct{}

func init() {
	viper.SetConfigFile(filepath.Join(utils.Basepath, "..", "gh-edu", "data", "data.json"))
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error with configuration file: %s\nRoot: %s", err.Error(), utils.Basepath)
	}
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true
	rootCmd.Flags().BoolVarP(&areTemplateF, "template", "t", false, "Indicate if there is a tutor template")
	rootCmd.Flags().VarP(&languageF, "language", "l", "Select the language. You can treat this flag as a boolean or pass a string")
	rootCmd.Flags().BoolVarP(&anonymizeF, "anonymize", "a", false, "Indicate if you want to randomize the names")
	rootCmd.Flags().BoolVarP(&quietF, "quiet", "q", false, "No INFO in the output only the result")
	rootCmd.Flags().StringVarP(&outputF, "output", "o", "", "Save the results in the specified path")
	rootCmd.Flags().IntVarP(&percentageF, "percentage", "p", 1, "Minimum porcentage to show links")
	rootCmd.Flags().IntVarP(&minLinesF, "min-lines", "m", 1, "Minimum lines to show links")
	rootCmd.Flags().StringVarP(&exerciseF, "exercise", "e", "", "Specify the regex for the assignment/exercise")
	viper.BindPFlag("assignment", rootCmd.Flags().Lookup("exercise"))
	rootCmd.Flags().StringVarP(&courseF, "course", "c", "", "Specify the course/organization")
	viper.BindPFlag("defaultOrg", rootCmd.Flags().Lookup("course"))
}

var (
	rootCmd = &cobra.Command{
		// Use:   "gh edu plagiarism [-a] [-q] [-l [<language>]] [-t]",
		Use:   "gh edu plagiarism",
		Short: "Detect plagiarism in students assignment",
		Long:  "gh-edu-plagiarism checks all the repositories related to an assignment and compares them to detect plagiarism",
		RunE:  func(cmd *cobra.Command, args []string) error { return realMain() },
	}
	areTemplateF bool
	languageF    langType
	outputF      string
	percentageF  int
	minLinesF    int
	exerciseF    string
	courseF      string
	anonymizeF   bool
	quietF       bool

	defaultOrgG string
	assignmentG string
	fzfOpMsgG   string
)

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
		for i, err := range errS {
			fmt.Println(i+1, ".", err)
		}
		return fmt.Errorf("Exiting with failure status due to previous errors")
	}
	err := cleanUp()
	if err != nil {
		return err
	}
	// Send request to GitHub API
	bufferLen, allRepos, err := request()
	if err != nil {
		return err
	}

	sendToCloneC := make(chan repoObj, bufferLen)
	selectTemplateC, selectedTemplateC := func() (chan string, chan string) { // TODO change logic don't use nil channels
		if areTemplateF {
			return make(chan string, bufferLen), make(chan string, 1)
		}
		return nil, nil
	}()
	errC := make(chan error)

	go filter(allRepos, sendToCloneC, selectTemplateC, errC)
	selectLangC := make(chan string, 1)
	go func() { // fzf (optional)
		selectLanguage(selectLangC, errC)
		selectTemplate(selectTemplateC, selectedTemplateC, errC)
	}()
	clonedC := make(chan repoObj)
	go clone(sendToCloneC, clonedC, errC)
	go send(clonedC, selectedTemplateC, selectLangC, errC) // send also close errC
	return <-errC
}

// check everything is installed, the server is up and set up globals
func check() []error {
	utils.Println(quietF, "Checking everything is ok...")
	if _, err := exec.LookPath("fzf"); err != nil {
		fzfOpMsgG = "No fzf command found. Indicate the value in the CLI or install fzf:\nhttps://github.com/junegunn/fzf"
	}
	mossPath := fmt.Sprintf("%s/moss", utils.Basepath)
	dependencies := map[string]string{
		// "fzf":    "You need to have fzf installed\nhttps://github.com/junegunn/fzf",
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
	// pinger, err := ping.NewPinger("moss.stanford.edu")
	// if err != nil {
	// 	errorS = append(errorS, fmt.Errorf("internal error: setting up ping: %w", err))
	// }
	// pinger.Count = 2
	// pinger.Timeout = time.Second * 2
	// err = pinger.Run() // Blocks until finished.
	// if err != nil {
	// 	errorS = append(errorS, err)
	// }
	// if pinger.Statistics().PacketsRecv == 0 {
	// 	errorS = append(errorS, fmt.Errorf("couldn't connect to the server"))
	// }
	if defaultOrgG = viper.GetString("defaultOrg"); defaultOrgG == "" {
		errorS = append(errorS, fmt.Errorf("please set an organization"))
	}
	if assignmentG = viper.GetString("assignmentR"); assignmentG == "" {
		errorS = append(errorS, fmt.Errorf("please set a current exercise/assignment"))
	}
	return errorS
}

// Clean up delete all the directories left by the last execution
// Is not done in the same execution because is uncertain how long the user
// is going to need the files with commands like xdg-open
func cleanUp() error {
	tempDir := os.TempDir()
	if tempDir == "" {
		return fmt.Errorf("internal error: cleanUp: couldn't access temp dir")
	}
	pattern := tempDir + `/*gh-edu-plagiarism`
	filesString, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("internal error: cleanUp: %w", err) // error access temporal directory
	}
	for _, fString := range filesString {
		err = os.RemoveAll(fString)
		if err != nil {
			fmt.Println("internal warning: cleanUp: couldn't delete: ", fString)
		}
	}
	return nil
}

// request return an optimal size for channels based on the number of members (Min(50, members))
// and all the repositories in the organization in JSON form
func request() (int, []string, error) {
	filter := []string{"--jq", ".data.organization.membersWithRole.totalCount, .data.organization.repositories.edges[].node"}
	response := utils.ExecuteQuery(utils.AllReposQ(defaultOrgG), filter...)
	responseSlice := strings.Split(response, "\n")
	membersN, err := strconv.Atoi(responseSlice[0])
	if err != nil {
		return 0, nil, fmt.Errorf("request: convert number to string: %w", err)
	}
	return utils.Min(50, membersN), responseSlice[1 : len(responseSlice)-1], nil
}

var langOptions = [...]string{"c", "cc", "java", "ml", "pascal", "ada", "lisp", "scheme", "haskell", "fortran", "ascii", "vhdl", "perl", "matlab", "python", "mips", "prolog", "spice", "vb", "csharp", "modula2", "a8086", "javascript", "plsql", "verilog"}

type langType string

func (l langType) String() string {
	return string(l)
}

func (l *langType) Set(v string) error {
	contain := func(options []string, value string) bool {
		for _, o := range options {
			if value == o {
				return true
			}
		}
		return false
	}
	if contain(langOptions[:], v) {
		*l = langType(v)
		return nil
	}
	return fmt.Errorf("must be one of: %v", langOptions)
}

func (e langType) Type() string {
	return "https://github.com/gh-cli-for-education/gh-edu-plagiarism#compatible-languages"
}

func selectLanguage(selectedLangC chan<- string, errC chan<- error) {
	if languageF != "" {
		selectedLangC <- string(languageF)
		return
	}
	if fzfOpMsgG != "" {
		errC <- fmt.Errorf(fzfOpMsgG)
		return
	}
	stdInFunc := func(in io.Writer) {
		for _, l := range langOptions {
			io.WriteString(in, l+"\n")
		}
	}
	languageS, err := utils.ExecuteCmd(utils.FzfCmd("Choose the programing language"), true, stdInFunc)
	if err != nil {
		errC <- fmt.Errorf("selectLanguage: %w", err)
		return
	}
	selectedLangC <- languageS
	close(selectedLangC)
}

func selectTemplate(reposC <-chan string, selectedTemplateC chan<- string, errC chan<- error) {
	if reposC == nil || selectedTemplateC == nil {
		return
	}
	if fzfOpMsgG != "" {
		errC <- fmt.Errorf(fzfOpMsgG)
		return
	}
	stdInFunc := func(in io.Writer) {
		for repo := range reposC {
			io.WriteString(in, repo+"\n")
		}
	}
	result, err := utils.ExecuteCmd(utils.FzfCmd("Select which repository is the template"), true, stdInFunc)
	if err != nil {
		errC <- fmt.Errorf("selectTemplate: %w", err)
		return
	}
	selectedTemplateC <- result
	close(selectedTemplateC)
}
