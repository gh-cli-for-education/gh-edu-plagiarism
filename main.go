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
			errS := check()
			if len(errS) > 0 {
				for _, err := range errS {
					fmt.Println(err)
				}
				os.Exit(1)
			}
			sendToCloneC := make(chan repoObj)
			selectTemplateC, selectedTemplateC := func() (chan string, chan string) {
				if areTemplate {
					return make(chan string), make(chan string)
				}
				return nil, nil
			}()
			go filter(sendToCloneC, selectTemplateC)
			go getTemplate(selectTemplateC, selectedTemplateC)
			clonedC := make(chan repoObj)
			removeC := make(chan empty)
			go clone(sendToCloneC, clonedC, removeC)
			send(clonedC, selectedTemplateC)
			removeC <- empty{}
			<-removeC
		},
	}
	areTemplate bool
)

func init() {
	viper.SetConfigFile("../gh-edu/config.json")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error with configuration file: " + err.Error())
	}
	rootCmd.Flags().BoolVarP(&areTemplate, "template", "t", false, "Indicate if there is a tutor template")
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
	result, err := executeCmd("fzf", true, stdInFunc)
	if err != nil {
		fmt.Println(err)
	}
	selectedTemplateC <- result
}

func clone(reposC <-chan repoObj, clonedReposC chan<- repoObj, remove chan empty) {
	dir, err := os.MkdirTemp("", "*_gh-edu-plagiarism")
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
	var builder strings.Builder
	regexUrl, _ := regexp.Compile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)

	clonedRepo := <-clonedReposC
	dir := string(regexp.MustCompile(".*/").Find([]byte(clonedRepo.dir)))
	builder.WriteString(fmt.Sprintf("%s/* ", clonedRepo.dir))
	for clonedRepo := range clonedReposC {
		builder.WriteString(fmt.Sprintf("%s/* ", clonedRepo.dir))
	}
	template := ""
	if selectedTemplateC != nil {
		template = fmt.Sprintf("-b %s%s/* ", dir, <-selectedTemplateC)
	}
	// Send request to Moss service
	mossCmd := fmt.Sprintf("./moss -l javascript -d %s %s", template, builder.String())
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
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
