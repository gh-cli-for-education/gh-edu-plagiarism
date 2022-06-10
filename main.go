package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type empty = struct{}

func allRepos(org string) string {
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

func init() {
	viper.SetConfigFile("../gh-edu/config.json")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error with configuration file: " + err.Error())
	}
}

type repoObj struct {
	Name string
	Url  string
	dir  string
}

var (
	rootCmd = &cobra.Command{
		Use:   "gh edu plagiarism",
		Short: "Detect plagiarism in students assigment",
		Long:  "gh-edu-plagiarism checks all the repositories from an assignment and compares it to detect plagiarism",
		Run: func(cmd *cobra.Command, args []string) {
			filtered2CloneC := make(chan repoObj)   // To clone
			filtered2TemplateC := make(chan string) // To select template
			selectedTemplateC := make(chan string)  // 1
			go filter(filtered2CloneC, filtered2TemplateC)
			go getTemplate(filtered2TemplateC, selectedTemplateC)
			clonedReposC := make(chan repoObj)
			remove := make(chan empty)
			go clone(filtered2CloneC, clonedReposC, remove)
			send(clonedReposC, selectedTemplateC)
			remove <- empty{}
			<-remove
		},
	}
)

func getTemplate(reposC <-chan string, selectedTemplateC chan<- string) {
	stdInFunc := func(in io.Writer) {
		for repo := range reposC {
			io.WriteString(in, repo+"\n")
		}
	}
	result, err := executeCmd("fzf", true, stdInFunc)
	if err != nil {
		fmt.Println(err)
	}
	selectedTemplateC <- result
}

func clone(reposC <-chan repoObj, clonedReposC chan<- repoObj, remove chan empty) {
	dir, err := os.MkdirTemp("", "*-gh_edu_plagiarism")
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
			repoDir := filepath.Join(dir, repo.Name)
			command := fmt.Sprintf("gh repo clone %s %s/", repo.Url, repoDir)
			_, err := executeCmd(command, false, nil)
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
	// When the signal is received clean up all the repositories and send another signal to let know it has finished
	<-remove
	err = os.RemoveAll(dir)
	if err != nil {
		fmt.Println(err)
	}
	remove <- empty{}
}

func send(clonedReposC <-chan repoObj, selectedTemplateC <-chan string) {
	// Set up
	selectedTemplate := ""
	var builder strings.Builder
	regexUrl, _ := regexp.Compile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)

	if selectedTemplateC != nil {
		selectedTemplate = <-selectedTemplateC
	}
	for clonedRepo := range clonedReposC {
		if clonedRepo.Name != selectedTemplate {
			builder.WriteString(fmt.Sprintf("%s/* ", clonedRepo.dir))
		}
	}
	// Send request to Moss service
	mossCmd := fmt.Sprintf("./moss -l javascript -d %s", builder.String())
	mossResult, err := executeCmd(mossCmd, false, nil)
	if err != nil {
		log.Println(err)
	}
	mossUrl := regexUrl.Find([]byte(mossResult))

	// Process the result with mossum TODO check more options in mossum
	mossumCmd := fmt.Sprintf("mossum -p 5 -r %s", mossUrl)
	mossumResult, err := executeCmd(mossumCmd, false, nil)
	if err != nil {
		log.Println(err)
	}
	fmt.Println("File:", mossumResult)
}

func main() {
	errS := check()
	if len(errS) > 0 {
		for _, err := range errS {
			fmt.Println(err)
		}
		os.Exit(1)
	}
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
