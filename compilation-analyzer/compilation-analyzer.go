package compilation_analyzer

import (
	"log"
	"fmt"
	"os"
	"bufio"
	"strings"
	"regexp"
)

type compilationAnalyzer struct {
	filename string
	compiler string
	linker string
	fixedPathPrefix string

	File *os.File
}

func SetupAnalyzer (filename, compiler, linker, fixedPath string) *compilationAnalyzer{
	var ana = new(compilationAnalyzer)

	ana.filename = filename
	ana.compiler = compiler
	ana.linker = linker
	ana.fixedPathPrefix = fixedPath

	file, err := os.Open(ana.filename)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	ana.File = file

	return ana
}

func (ana compilationAnalyzer) ProcessFile() {
	outputFile, err := os.Create("Output.dot")
	defer outputFile.Close()

	if err != nil {
		log.Fatal(err)
		return
	}

	outputFile.WriteString("digraph G {\n")

	fmt.Println("Processing", ana.filename)
	objects := regexp.MustCompile(`(?m)-o ..((?:\/[\w\.\-]+)+)`)
	sources := regexp.MustCompile(`(?m)-c ..((?:\/[\w\.\-]+)+)`)
	//paths := regexp.MustCompile(`..((?:\/[\w\.\-]+)+)`)
	cd := regexp.MustCompile(`((?:\/[\w\.\-]+)+)`)

	scanner := bufio.NewScanner(ana.File)
	var prefix string

	for scanner.Scan() {
		if text := scanner.Text(); strings.Contains(text, "cd"){
			prefix = cd.FindAllString(text,-1)[0]
			tmp := strings.Replace(prefix, ana.fixedPathPrefix,"",-1)
			tmp = strings.Replace(tmp,"Make","",-1)
			prefix = tmp
		}

		if text := scanner.Text(); strings.Contains(text, ana.compiler) {
			objsStr := objects.FindAllString(text, 1)
			objStr := strings.Split(objsStr[0], " ")[1]
			objStr = strings.Replace(objStr,"../",prefix,-1)


			srcs_str := sources.FindAllString(text, -1)
			for _, match := range srcs_str {
				src_str := strings.Split(match, " ")[1]
				src_str = strings.Replace(src_str,"../",prefix,-1)
				outputFile.WriteString("\t\"" + src_str + "\" -> \"" + objStr + "\";\n")
			}
		}

		if text := scanner.Text(); strings.Contains(text, ana.linker) {
			objsStr := objects.FindAllString(text, 1)
			objStr := strings.Split(objsStr[0], " ")[1]
			objStr = prefix + objStr
			objStr = strings.Replace(objStr, ana.fixedPathPrefix,"",-1)
			objStr = strings.Replace(objStr,"../","",-1)

			pathsStr := cd.FindAllString(text, -1)
			for i, match := range pathsStr {
				if i == 0 {
					continue
				}

				tmp := strings.Replace(match, ana.fixedPathPrefix,"",-1)

				if !strings.Contains(tmp,prefix) {
					if !strings.Contains(match, ana.fixedPathPrefix) {
						tmp = prefix + tmp
						tmp = strings.Replace(tmp,"//","/",-1)
					}
				}
				fmt.Println(tmp)

				outputStr := "\t\"" + tmp + "\" -> \"" + objStr + "\";\n"
				outputFile.WriteString(outputStr)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	outputFile.WriteString("}\n")

}