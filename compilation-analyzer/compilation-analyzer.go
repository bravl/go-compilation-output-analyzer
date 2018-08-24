package compilation_analyzer

import (
	"log"
	"fmt"
	"os"
	"bufio"
	"strings"
	"regexp"
	"os/exec"
	"io/ioutil"
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

func figureOutComponent(input string) string{
	results := strings.Split(input,"/")
	return results[0]
}

func figureOutFunc(input string) string{
	if !strings.Contains(input,"!_") {
		tmpFunctionString := strings.Split(input,"/^")[1]
		tmpFunctionString = strings.Split(tmpFunctionString,"$/")[0]
		return tmpFunctionString
	}
	return ""
}

func figureOutFilename(input,prefix string) string{
	if !strings.Contains(input,"!_") {
		tmpFileString:= strings.Replace(input,prefix,"",-1)
		tmpFileString = strings.SplitN(tmpFileString,"\t",-1)[1]
		return tmpFileString
	}
	return ""
}

func (ana compilationAnalyzer) figureOutFunctions(filename string, sourcesFile *os.File) {
	if _, err := os.Stat(ana.sourcePath+filename); os.IsNotExist(err) {
		fmt.Println("Failed\n")
	}
	cmd := exec.Command("ctags","--c-types=f","--if0=no","-o tmptags", ana.sourcePath+filename)
	cmd.Run()

	tmpTagsFile,err := os.Open("tmptags")
	if err != nil {
		log.Fatal(err)
		return
	}

	scanner := bufio.NewScanner(tmpTagsFile)

	for scanner.Scan() {
		tmpString := figureOutFunc(scanner.Text())
		if tmpString != "" {
			sourcesFile.WriteString("\t\t" + tmpString + "\n")
		}
	}

	os.Remove("tmptags")
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

func (ana compilationAnalyzer) CreateCscopeCtagsDB() {
	scriptFile,_ := os.Create("gen_helper_files.sh")
	scriptFile.WriteString("find "+strings.Replace(ana.sourcePath," ","\\ ",-1)+" -name \"*.c\" > cscope.files\n")
	scriptFile.WriteString("ctags -R --c-types=f -o alltags "+strings.Replace(ana.sourcePath," ","\\ ", -1))

	cmd := exec.Command("sh","./gen_helper_files.sh")
	cmd.Run()
}

func (ana compilationAnalyzer) ProcessTags() {
	sourcesData,_ := ioutil.ReadFile("sources.txt")
	file,_ := os.Open("alltags")
	scanner := bufio.NewScanner(file)

	unusedFuncs := 0
	usedFuncs := 0
	emptyLines := 0

	used,_ := os.Create("usedfuncs.txt")
	unused,_ := os.Create("unusedfuncs.txt")
	usedData,_ := ioutil.ReadFile("usedfuncs.txt")
	unusedData,_ := ioutil.ReadFile("unusedfuncs.txt")

	for scanner.Scan() {
		fileName := figureOutFilename(scanner.Text(),ana.sourcePath)
		funcName := figureOutFunc(scanner.Text())
		if funcName == "" {
			emptyLines++
			continue
		}
		if strings.Contains(string(sourcesData), funcName) && !strings.Contains(string(usedData), funcName) {
			usedFuncs++
			used.WriteString(funcName+"\n");
			usedData,_ = ioutil.ReadFile("usedfuncs.txt")
		} else {
			if !strings.Contains(string(unusedData), funcName) {
				unusedFuncs++
				unused.WriteString(fileName + " - \t" + funcName+"\n");
				unusedData,_ = ioutil.ReadFile("unusedfuncs.txt")
			}
		}
	}

	fmt.Println("Used functions:",usedFuncs)
	fmt.Println("Unused functions:",unusedFuncs)
	fmt.Println("Empty lines:",emptyLines)

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