package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/gh-cli-for-education/gh-edu-plagiarism/pkg/utils"
)

func send(clonedReposC <-chan repoObj, selectedTemplateC <-chan string, selectedLangC <-chan string, errC chan<- error) {
	// Set up
	regexUrl, err := regexp.Compile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)
	if err != nil {
		errC <- fmt.Errorf("internal error: send: regex to get URL from MOSS serve: %w", err)
	}
	var builder strings.Builder

	clonedRepo := <-clonedReposC                                             // Get temp directory reading from the first cloned repo
	tmpDir := string(regexp.MustCompile(".*/").Find([]byte(clonedRepo.dir))) // TODO should I panic?
	builder.WriteString(fmt.Sprintf("%s/* ", clonedRepo.dir))
	for clonedRepo := range clonedReposC {
		builder.WriteString(fmt.Sprintf("%s/* ", clonedRepo.dir))
	}
	template := ""
	if selectedTemplateC != nil {
		template = fmt.Sprintf("-b %s%s/* ", tmpDir, <-selectedTemplateC)
	}
	// Send request to Moss service
	mossCmd := fmt.Sprintf("%s/moss -l %s -d %s %s", utils.Basepath, <-selectedLangC, template, builder.String()) // TODO let the user decide lines or % TODO --output flag
	utils.Println(quietF, "Connecting with Moss server...")
	mossResult, err := utils.ExecuteCmd(mossCmd, false, nil)
	if err != nil {
		log.Println(err)
	}
	mossUrl := regexUrl.Find([]byte(mossResult))
	process(mossUrl, tmpDir, errC)
	close(errC)
}

// Process the result with mossum. TODO check more options in mossum
func process(mossUrl []byte, tmpDir string, errC chan<- error) {
	aF := ""
	if anonymizeF {
		aF = "-a"
	}
	mossumCmd := fmt.Sprintf("mossum -p 5 -r -t \".*/(.+)/.*\" %s -o %s/result %s", aF, tmpDir, mossUrl) // .*/(.+).* from <randNumber>gh-edu-plagiarism/assigmentName/ get assigmentName
	utils.Println(quietF, "Generating graph...")
	_, err := utils.ExecuteCmd(mossumCmd, false, nil)
	if err != nil {
		fmt.Println("Something went wrong with mossum. Here is the Moss URL:", string(mossUrl))
		errC <- fmt.Errorf("mossum: %w", err)
		return
	}
	f, err := os.Open(tmpDir + "/result.txt")
	if err != nil {
		fmt.Println("error: mossum: couldn't open report file\n", err)
	} else {
		// ReadAll will return a slice as big as the file generated but mossum.
		// renember this is n!/((n-2)!2!) pairs. If the file is too big use io.Copy
		report, err := io.ReadAll(f)
		if err != nil {
			fmt.Println("error: mossum: couldn't read report file\n", err)
		} else {
			utils.Println(quietF, "Report:")
			fmt.Print(string(report)) // For some reason mossum report has 2 extra lines
		}
	}
	if err = utils.OpenFile(tmpDir + "result-1.png"); err != nil {
		fmt.Println("error: mossum: couldn't open graph image\n", err)
	}
}
