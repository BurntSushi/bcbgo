package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"sync"

	"github.com/BurntSushi/bcbgo/cmd/util"
	// "github.com/BurntSushi/bcbgo/io/pdb"
	"github.com/BurntSushi/bcbgo/io/pdb/slct"
)

var (
	flagOverwrite = false
	flagPdbSelect = false
)

func init() {
	flag.BoolVar(&flagOverwrite, "overwrite", flagOverwrite,
		"When set, any existing database will be completely overwritten.")
	flag.BoolVar(&flagPdbSelect, "pdb-select", flagPdbSelect,
		"When set, the input will be read as a PDB Select file.")

	util.FlagUse("cpu", "cpuprof", "verbose")
	util.FlagParse(
		"frag-lib-path protein-list fseqdb-out-file",
		"Where 'protein-list' is a plain text file with PDB chain\n"+
			"identifiers on each line. e.g., '1P9GA'.")
	util.AssertLeastNArg(3)
}

func main() {
	libPath := util.Arg(0)
	protList := util.Arg(1)
	dbPath := util.Arg(2)

	_ = util.FragmentLibrary(libPath)
	if flagOverwrite {
		util.Assert(os.RemoveAll(dbPath),
			"Could not remove '%s' directory", dbPath)
	}
	if len(util.FlagCpuProf) > 0 {
		f := util.CreateFile(util.FlagCpuProf)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	wg := new(sync.WaitGroup)
	chainIds := genChains(protList)
	for i := 0; i < util.FlagCpu; i++ {
		wg.Add(1)
		go func() {
			for chainId := range chainIds {
				_, chain := util.PDBReadId(chainId)
				fmt.Printf("%s%c\n", chain.Entry.IdCode, chain.Ident)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func genChains(protList string) chan string {
	ids := make([]string, 0, 100)
	file := util.OpenFile(protList)
	if flagPdbSelect {
		records, err := slct.NewReader(file).ReadAll()
		util.Assert(err)
		for _, r := range records {
			if len(r.ChainID) != 5 {
				util.Fatalf("Not a valid chain identifier: '%s'", r.ChainID)
			}
			ids = append(ids, r.ChainID)
		}
	} else {
		for _, line := range util.ReadLines(file) {
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			} else if len(line) != 5 {
				util.Fatalf("Not a valid chain identifier: '%s'\n"+
					"Perhaps you forgot to set 'pdb-select'?", line)
			}
			ids = append(ids, line)
		}
	}

	// Convert chain IDs to a channel.
	// Idea: multiple goroutines can read and parse PDB files in parallel.
	chains := make(chan string)
	go func() {
		for _, id := range ids {
			chains <- id
		}
		close(chains)
	}()
	return chains
}
