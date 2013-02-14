package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"

	"github.com/BurntSushi/bcbgo/bow"
	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

var (
	flagOverwrite = false
)

func init() {
	flag.BoolVar(&flagOverwrite, "overwrite", flagOverwrite,
		"When set, any existing database will be completely overwritten.")

	util.FlagUse("cpu", "cpuprof", "verbose", "seq-db", "pdb-hhm-db", "blits",
		"hhfrag-min", "hhfrag-max", "hhfrag-inc")
	util.FlagParse(
		"bowdb-path frag-lib-path "+
			"(protein-dir | (protein-file [protein-file ...]))",
		"Where a protein file is a FASTA, fmap or PDB file.")
	util.AssertLeastNArg(3)
}

func main() {
	dbPath := util.Arg(0)
	libPath := util.Arg(1)
	fileArgs := flag.Args()[2:]

	if flagOverwrite {
		util.Assert(os.RemoveAll(dbPath),
			"Could not remove '%s' directory", dbPath)
	}

	db, err := bow.CreateDB(util.FragmentLibrary(libPath), dbPath)
	util.Assert(err)

	if len(util.FlagCpuProf) > 0 {
		f := util.CreateFile(util.FlagCpuProf)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	files := util.AllFilesFromArgs(fileArgs)
	progress := util.NewProgress(len(files))

	fileChan := make(chan string)
	wg := new(sync.WaitGroup)
	for i := 0; i < max(1, runtime.GOMAXPROCS(0)); i++ {
		go func() {
			wg.Add(1)
			for file := range fileChan {
				addToDB(db, file, progress)
			}
			wg.Done()
		}()
	}

	for _, file := range files {
		fileChan <- file
	}

	close(fileChan)
	wg.Wait()
	progress.Close()
	util.Assert(db.Close())
}

func addToDB(db *bow.DB, file string, progress util.Progress) {
	if util.IsFasta(file) || util.IsFmap(file) {
		fmap := util.GetFmap(file)
		db.Add(fmap)
		progress.JobDone(nil)
	} else if util.IsPDB(file) {
		entry, err := pdb.ReadPDB(file)
		if err != nil {
			progress.JobDone(fmt.Errorf(
				"Error reading PDB file '%s': %s", file, err))
			return
		}
		for _, chain := range entry.Chains {
			if chain.IsProtein() {
				db.Add(chain)
			}
		}
		progress.JobDone(nil)
	} else {
		progress.JobDone(fmt.Errorf("Unrecognized protein file: '%s'.", file))
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
