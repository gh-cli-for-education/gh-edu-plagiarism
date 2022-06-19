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

func send(clonedReposC <-chan repoObj, selectedTemplateC <-chan string, errC chan<- error) {
	// Set up
	var builder strings.Builder
	regexUrl, err := regexp.Compile(`https?:\/\/(www\.)?[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`)
	if err != nil {
		errC <- fmt.Errorf("internal error: send: regex to get URL from MOSS serve: %w", err)
	}

	clonedRepo := <-clonedReposC                                             // Get temp directory reading from the first cloned repo
	tmpDir := string(regexp.MustCompile(".*/").Find([]byte(clonedRepo.dir))) // TODO should I panic?
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
			for _, l := range langOptions {
				io.WriteString(in, l+"\n")
			}
		}
		languageS, err := utils.ExecuteCmd("fzf", true, stdInFunc)
		if err != nil {
			fmt.Println(err)
		}
		language = langType(languageS)
	}
	mossCmd := fmt.Sprintf("%s/moss -l %s -d %s %s", utils.Basepath, language, template, builder.String())
	utils.Println(quiet, "Connecting with Moss server...")
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
	if anonymize {
		aF = "-a"
	}
	mossumCmd := fmt.Sprintf("mossum -p 5 -r -t \".*/(.+)/.*\" %s -o %s/result %s", aF, tmpDir, mossUrl) // .*/(.+).* from <randNumber>gh-edu-plagiarism/assigmentName/ get assigmentName
	utils.Println(quiet, "Generating graph...")
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
			utils.Println(quiet, "Report:")
			fmt.Print(string(report)) // For some reason mossum report has 2 extra lines
		}
	}
	if err = utils.OpenFile(tmpDir + "result-1.png"); err != nil {
		fmt.Println("error: mossum: couldn't open graph image\n", err)
	}
}
