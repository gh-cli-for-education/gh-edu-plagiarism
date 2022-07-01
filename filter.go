package main

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
)

func filter(allRepos []string, repos2CloneC chan<- repoObj, selectTemplateC chan<- string, errC chan<- error) {
	regex, err := regexp.Compile(assignmentG)
	if err != nil {
		errC <- fmt.Errorf("filter: assignment regex is not valid: %w", err)
	}
	if selectTemplateC == nil {
		filterReposNoTemplate(allRepos, regex, repos2CloneC, errC)
	} else {
		filterReposWithTemplate(allRepos, regex, repos2CloneC, selectTemplateC, errC)
	}
}

func filterReposNoTemplate(allRepos []string, regex *regexp.Regexp, repos2CloneC chan<- repoObj, errC chan<- error) {
	for _, repo := range allRepos {
		var obj repoObj
		if err := json.Unmarshal([]byte(repo), &obj); err != nil {
			log.Panicf("filter(no template): parse json: %s \nwith repo: %s", err, repo)
		}
		if regex.Match([]byte(obj.Name)) {
			repos2CloneC <- obj
		}
	}
	close(repos2CloneC)
}

func filterReposWithTemplate(allRepos []string, regex *regexp.Regexp, repos2CloneC chan<- repoObj, selectTemplateC chan<- string, errC chan<- error) {
	for _, repo := range allRepos {
		var obj repoObj
		if err := json.Unmarshal([]byte(repo), &obj); err != nil {
			log.Panicf("filter(no template): parse json: %s \nwith repo: %s", err, repo)
		}
		if regex.Match([]byte(obj.Name)) {
			selectTemplateC <- obj.Name
			repos2CloneC <- obj
		}
	}
	close(selectTemplateC)
	close(repos2CloneC)
}
