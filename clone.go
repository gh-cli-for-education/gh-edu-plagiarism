package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/gh-cli-for-education/gh-edu-plagiarism/pkg/utils"
)

func clone(reposC <-chan repoObj, clonedReposC chan<- repoObj, errC chan<- error) {
	tmpDir, err := os.MkdirTemp("", "*_gh-edu-plagiarism")
	if err != nil {
		errC <- fmt.Errorf("clone: creating tmp dir, %w", err)
		return
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
				errC <- fmt.Errorf("clone: %w", err)
				return
			}
			repo.dir = repoDir
			clonedReposC <- repo
			<-sem
		}(repo)
	}
	wg.Wait()
	close(clonedReposC)
}
