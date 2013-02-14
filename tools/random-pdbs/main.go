package main

import (
	"flag"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/BurntSushi/bcbgo/cmd/util"
)

var (
	flagNum   = 10
	flagPaths = false
)

func init() {
	flag.IntVar(&flagNum, "n", flagNum, "The number of PDB entries to echo.")
	flag.BoolVar(&flagPaths, "paths", flagPaths,
		"When set, full file paths will be echoed instead of PDB ids.")

	util.FlagUse("pdb-dir")
	util.FlagParse("", "")

	rand.Seed(time.Now().UnixNano())
}

func main() {
	pdbFiles := util.RecursiveFiles(util.FlagPdbDir)
	files := make([]string, 0, flagNum)
	for i := 0; i < flagNum; i++ {
		var index int = -1
		for index == -1 || !util.IsPDB(pdbFiles[index]) {
			// not guaranteed to terminate O_O
			index = rand.Intn(len(pdbFiles))
		}
		files = append(files, pdbFiles[index])
		pdbFiles = append(pdbFiles[:index], pdbFiles[index+1:]...)
	}

	for _, f := range files {
		if flagPaths {
			fmt.Println(f)
		} else {
			e := util.PDBRead(f)
			fmt.Printf("%s%c\n", strings.ToLower(e.IdCode), e.Chains[0].Ident)
		}
	}
}
