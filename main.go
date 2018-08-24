package main

import (
	"./compilation-analyzer"
	"os/exec"
)

func main() {
	ana := compilation_analyzer.SetupAnalyzer("./make_ppc.ksh.build.log", "ccppc", "ldppc",
												"/export/home/powerpc/RCU_PPC.proj/")
	ana.ProcessFile()

	cmd := exec.Command("dot", "-Tsvg", "Output.dot" ,"-o Output.svg")
	cmd.Run()
}