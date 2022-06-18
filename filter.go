package main

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"github.com/gh-cli-for-education/gh-edu-plagiarism/pkg/utils"
	"github.com/spf13/viper"
)

func filter(repos2CloneC chan<- repoObj, selectTemplateC chan<- string) {
	regex, err := regexp.Compile(viper.GetString("assignment"))
	if err != nil {
		log.Panic(err)
	}
	filter := []string{"--jq", ".data.organization.repositories.edges[].node | {name, url}"}
  AllReposQ := utils.AllReposQ(viper.GetString("defaultOrg"))
	allRepos := strings.Split(utils.ExecuteQuery(AllReposQ, filter...), "\n")
	if selectTemplateC == nil {
		filterReposNoTemplate(allRepos, regex, repos2CloneC)
	} else {
		filterReposWithTemplate(allRepos, regex, repos2CloneC, selectTemplateC)
	}
}

func filterReposNoTemplate(allRepos []string, regex *regexp.Regexp, repos2CloneC chan<- repoObj) {
	for _, repo := range allRepos[:len(allRepos)-1] {
		var obj repoObj
		json.Unmarshal([]byte(repo), &obj)
		if regex.Match([]byte(obj.Name)) {
			repos2CloneC <- obj
		}
	}
	close(repos2CloneC)
}

func filterReposWithTemplate(allRepos []string, regex *regexp.Regexp, repos2CloneC chan<- repoObj, selectTemplateC chan<- string) {
	var waitingRepos []repoObj
	for _, repo := range allRepos[:len(allRepos)-1] {
		var obj repoObj
		json.Unmarshal([]byte(repo), &obj)
		if regex.Match([]byte(obj.Name)) {
			selectTemplateC <- obj.Name
			select {
			case repos2CloneC <- obj:
			default:
				waitingRepos = append(waitingRepos, obj)
			}
		}
	}
	close(selectTemplateC)
	for _, repo := range waitingRepos {
		repos2CloneC <- repo
	}
	close(repos2CloneC)
}
