package compilation_analyzer

import (
	"log"
	"fmt"
	"os"
	"bufio"
	"strings"
	"regexp"
	"os/exec"
)

var objects = regexp.MustCompile(`(?m)-o ..((?:\/[\w\.\-]+)+)`)
var sources = regexp.MustCompile(`(?m)-c ..((?:\/[\w\.\-]+)+)`)
var paths = regexp.MustCompile(`((?:\/[\w\.\-]+)+)`)

type compilationAnalyzer struct {
	filename string
	compiler string
	linker string
	fixedPathPrefix string
	sourcePath string

	File *os.File
}

func SetupAnalyzer (filename, compiler, linker, fixedPath, sourcePath string) *compilationAnalyzer{
	var ana = new(compilationAnalyzer)

	ana.filename = filename
	ana.compiler = compiler
	ana.linker = linker
	ana.fixedPathPrefix = fixedPath
	ana.sourcePath = sourcePath

	file, err := os.Open(ana.filename)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	ana.File = file

	return ana
}

func figureOutComponent(input string) string{
	results := strings.Split(input,"/")
	return results[0]
}

func (ana compilationAnalyzer) figureOutFunctions(filename string, sourcesFile *os.File) {
	if _, err := os.Stat(ana.sourcePath+filename); os.IsNotExist(err) {
		fmt.Println("Failed\n")
	}
	cmd := exec.Command("ctags","--c-types=f","-o tmptags", ana.sourcePath+filename)
	cmd.Run()

	tmpTagsFile,err := os.Open("tmptags")
	if err != nil {
		log.Fatal(err)
		return
	}

	scanner := bufio.NewScanner(tmpTagsFile)

	for scanner.Scan() {
		if !strings.Contains(scanner.Text(),"!_") {
			tmpFunctionString := strings.Split(scanner.Text(),"/^")[1]
			tmpFunctionString = strings.Split(tmpFunctionString,"$/")[0]
			sourcesFile.WriteString("\t\t"+tmpFunctionString+"\n")
		}
	}

	os.Remove("tmptags")
}

func (ana compilationAnalyzer) ProcessFileToDot() {
	outputFile, err := os.Create("Output.dot")
	defer outputFile.Close()

	if err != nil {
		log.Fatal(err)
		return
	}

	sourcesFile, err := os.Create("sources.txt")
	defer outputFile.Close()

	if err != nil {
		log.Fatal(err)
		return
	}

	outputFile.WriteString("digraph G {\n")

	fmt.Println("Processing", ana.filename)

	scanner := bufio.NewScanner(ana.File)
	var prefix string
	var oldComponent string

	for scanner.Scan() {
		if text := scanner.Text(); strings.Contains(text, "cd"){
			prefix = paths.FindAllString(text,-1)[0]
			tmp := strings.Replace(prefix, ana.fixedPathPrefix,"",-1)
			tmp = strings.Replace(tmp,"Make","",-1)
			prefix = tmp
			component := figureOutComponent(prefix)
			if component != oldComponent {
				sourcesFile.WriteString(component + "\n");
				oldComponent = component
			}

		}

		if text := scanner.Text(); strings.Contains(text, ana.compiler) {
			objsStr := objects.FindAllString(text, 1)
			objStr := strings.Split(objsStr[0], " ")[1]
			objStr = strings.Replace(objStr,"../",prefix,-1)

			srcsStr := sources.FindAllString(text, -1)
			for _, match := range srcsStr {
				srcStr := strings.Split(match, " ")[1]
				srcStr = strings.Replace(srcStr,"../",prefix,-1)
				outputFile.WriteString("\t\"" + srcStr + "\" -> \"" + objStr + "\";\n")
				tmp := strings.Split(srcStr,"/")
				sourcesFile.WriteString("\t" + tmp[len(tmp) - 1] + "\n")
				ana.figureOutFunctions(srcStr,sourcesFile)
			}
		}

		if text := scanner.Text(); strings.Contains(text, ana.linker) {
			objsStr := objects.FindAllString(text, 1)
			objStr := strings.Split(objsStr[0], " ")[1]
			objStr = prefix + objStr
			objStr = strings.Replace(objStr, ana.fixedPathPrefix,"",-1)
			objStr = strings.Replace(objStr,"../","",-1)

			pathsStr := paths.FindAllString(text, -1)
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