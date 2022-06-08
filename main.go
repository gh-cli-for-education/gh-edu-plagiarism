package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	// "github.com/spf13/viper"
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
  // viper.SetConfigFile()
  _, b, _, _ := runtime.Caller(0)
  fmt.Println(b)
}

type repoObj struct {
	Name string
	Url  string
	dir  string
}

var (
	defaultOrg = "gh-cli-for-education"
	rootCmd    = &cobra.Command{
		Use:   "gh edu plagiarism",
		Short: "Detect plagiarism in students assgiment",
		Long:  "gh-edu-plagiarism checks all the repositories from an assgiment and compares it to detect plagiarism",
		Run: func(cmd *cobra.Command, args []string) {
			regex, err := regexp.Compile("(testing)+.*")
			if err != nil {
				fmt.Println(err)
				return
			} // TODO create file to dump errors
			reposC := make(chan repoObj)
			go getRepos(regex, reposC)
			clonedReposC := make(chan repoObj)
			remove := make(chan empty)
			go clone(reposC, clonedReposC, remove)
			for clonedRepo := range clonedReposC {
				fmt.Printf("%+v\n", clonedRepo)
			}
			remove <- empty{}
			<-remove
		},
	}
)

func getRepos(regex *regexp.Regexp, reposC chan<- repoObj) {
	filter := []string{"--jq", ".data.organization.repositories.edges[].node | {name, url}"}
	allRepos := strings.Split(executeQuery(allRepos(defaultOrg), filter...), "\n")
	for _, repo := range allRepos[:len(allRepos)-1] {
		var obj repoObj
		json.Unmarshal([]byte(repo), &obj)
		if regex.Match([]byte(obj.Name)) {
			reposC <- obj
		}
	}
	close(reposC)
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
		go func(repo repoObj) { // TODO add smart output -> show how files are clone in a fixed place in the terminal
			defer wg.Done()
			repoDir := filepath.Join(dir, repo.Name)
			command := fmt.Sprintf("gh repo clone %s %s/", repo.Url, repoDir)
			_, err := executeCmd(command, true, nil)
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
