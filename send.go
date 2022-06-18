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

func send(clonedReposC <-chan repoObj, selectedTemplateC <-chan string) {
	// Set up
	var builder strings.Builder
	regexUrl, _ := regexp.Compile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)

	clonedRepo := <-clonedReposC // Get temp directory reading from the first cloned repo
	tmpDir := string(regexp.MustCompile(".*/").Find([]byte(clonedRepo.dir)))
	builder.WriteString(fmt.Sprintf("%s/* ", clonedRepo.dir))
	for clonedRepo := range clonedReposC {
		builder.WriteString(fmt.Sprintf("%s/* ", clonedRepo.dir))
	}
	template := ""
	if selectedTemplateC != nil {
		template = fmt.Sprintf("-b %s%s/* ", tmpDir, <-selectedTemplateC) // The concurrency ends here
	}
	// Send request to Moss service
	if language == "" {
		stdInFunc := func(in io.Writer) {
			for _, l := range utils.LangOptions {
				io.WriteString(in, l+"\n")
			}
		}
		languageS, err := utils.ExecuteCmd("fzf", true, stdInFunc)
		if err != nil {
			fmt.Println(err)
		}
		language = utils.Language(languageS)
	}
	mossCmd := fmt.Sprintf("%s/moss -l %s -d %s %s", utils.Basepath, language, template, builder.String())
	fmt.Println("Connecting with Moss server...")
	mossResult, err := utils.ExecuteCmd(mossCmd, false, nil)
	if err != nil {
		log.Println(err)
	}
	mossUrl := regexUrl.Find([]byte(mossResult))
	process(mossUrl, tmpDir)
}

// Process the result with mossum. TODO check more options in mossum
func process(mossUrl []byte, tmpDir string) {
  aF := ""
  if anonymize {
    aF = "-a"
  }
	mossumCmd := fmt.Sprintf("mossum -p 5 -r -t \".*/(.+)/.*\" %s -o %s/result %s", aF, tmpDir, mossUrl) // .*/(.+).* from <randNumber>gh-edu-plagiarism/assigmentName/ get assigmentName
	fmt.Println("Generating graph...")
	_, err := utils.ExecuteCmd(mossumCmd, false, nil)
	if err != nil {
		log.Println(err)
	}
	f, err := os.Open(tmpDir + "/result.txt")
	if err != nil {
		log.Println(err)
	}
  // ReadAll will return a slice as big as the file generated but mossum.
  // renember this is n!/((n-2)!2!) pairs. If the file is too big use io.Copy
	report, err := io.ReadAll(f)
	if err != nil {
		log.Println(err)
	}
	fmt.Print("Report:\n", string(report))
	utils.OpenFile(tmpDir + "result-1.png")
}
