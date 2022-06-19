package main

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/spf13/viper"
)

func filter(allRepos []string, repos2CloneC chan<- repoObj, selectTemplateC chan<- string, errC chan<- error) {
	regex, err := regexp.Compile(viper.GetString("assignment"))
	if err != nil {
		errC <- fmt.Errorf("filter: assignment regex: %w", err)
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
		err := json.Unmarshal([]byte(repo), &obj)
		if err != nil {
			errC <- fmt.Errorf("filter(no template): parse json: %w", err)
		}
		if regex.Match([]byte(obj.Name)) {
			repos2CloneC <- obj
		}
	}
	close(repos2CloneC)
}

func filterReposWithTemplate(allRepos []string, regex *regexp.Regexp, repos2CloneC chan<- repoObj, selectTemplateC chan<- string, errC chan<- error) {
	// var waitingRepos []repoObj
	for _, repo := range allRepos {
		var obj repoObj
		err := json.Unmarshal([]byte(repo), &obj)
		if err != nil {
			errC <- fmt.Errorf("filter(with template): parse json: %w", err)
		}
		if regex.Match([]byte(obj.Name)) {
			selectTemplateC <- obj.Name
			repos2CloneC <- obj
			// select { // Order is not important so queue the filtered repos if clone module is at full capacity
			// case repos2CloneC <- obj:
			// default:
			// waitingRepos = append(waitingRepos, obj)
			// }
		}
	}
	close(selectTemplateC)
	// for _, repo := range waitingRepos {
	// repos2CloneC <- repo
	// }
	close(repos2CloneC)
}
