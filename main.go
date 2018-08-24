package main

import (
	"./compilation-analyzer"
	"os/exec"
)

func main() {
	ana := compilation_analyzer.SetupAnalyzer("./make_ppc.ksh.build.log", "ccppc", "ldppc",
												"/export/home/powerpc/RCU_PPC.proj/",
												"/Users/bravl/Code/Thales/AWACS_RCU_CSCI BUILD_4.94_3SM 00003 AAAC/BUILD 4.94/CSCI_BUILD_4.94/RCU_PPC.proj/")
	ana.ProcessFileToDot()


	cmd := exec.Command("dot", "-Tsvg", "Output.dot" ,"-o Output.svg")
	cmd.Run()
}