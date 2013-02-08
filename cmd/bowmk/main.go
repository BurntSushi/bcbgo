// Example bag-of-words shows how to compute a bag-of-words vector given a
// fragment library and a PDB file.
package main

import (
	"flag"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

var (
	flagCpuProfile = ""
	flagOverwrite  = false
)

func init() {
	flag.StringVar(&flagCpuProfile, "cpuprofile", flagCpuProfile,
		"When set, a CPU profile will be written to the file provided.")
	flag.BoolVar(&flagOverwrite, "overwrite", flagOverwrite,
		"When set, any existing database will be completely overwritten.")

	util.FlagUse("cpu", "verbose")
	util.FlagParse(
		"bowdb-path frag-lib-path (pdb-dir | (pdb-file [pdb-file ...]))", "")
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

	pdbFiles := make([]string, 0)
	for _, fordir := range fileArgs {
		var more []string
		if util.IsDir(fordir) {
			more = util.RecursiveFiles(fordir)
		} else {
			more = []string{fordir}
		}
		pdbFiles = append(pdbFiles, more...)
	}

	lib := util.FragmentLibrary(libPath)
	util.Verbosef("Using library %s.\n", lib)

	db, err := bowdb.Create(lib, dbPath)
	util.Assert(err)

	pool := newBowWorkers(lib, max(1, runtime.GOMAXPROCS(0)))
	progressChan, doneProgress := progress(len(pdbFiles))
	doneWriting, err := writer(db, progressChan, pool)
	util.Assert(err)

	if len(flagCpuProfile) > 0 {
		f := util.CreateFile(flagCpuProfile)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	for _, pdbFile := range pdbFiles {
		entry, err := pdb.ReadPDB(pdbFile)
		if err != nil {
			progressChan <- progressJob{pdbFile, err}
			continue
		}

		// If there aren't any protein chains, skip this entry but update
		// the progress bar.
		hasProteinChains := false
		for _, chain := range entry.Chains {
			if chain.IsProtein() {
				hasProteinChains = true
				break
			}
		}
		if !hasProteinChains {
			progressChan <- progressJob{pdbFile, nil}
			continue
		}
		pool.enqueue(entry)
	}

	pool.done()
	<-doneWriting
	close(progressChan)
	<-doneProgress
	util.Verbosef("\n")

	util.Assert(db.WriteClose())
}

func writer(db *bowdb.DB,
	pchan chan progressJob, pool pool) (chan struct{}, error) {

	done := make(chan struct{}, 0)
	go func() {
		for result := range pool.results {
			pchan <- progressJob{result.chain.Entry.Path, nil}
			util.Assert(db.Write(result.chain, result.bow))
		}
		done <- struct{}{}
	}()
	return done, nil
}

type progressJob struct {
	path string
	err  error
}

func progress(total int) (chan progressJob, chan struct{}) {
	pchan := make(chan progressJob, 15)
	successCnt, errCnt := 0, 0
	counted := make(map[string]struct{}, 100)
	done := make(chan struct{}, 0)
	go func() {
		for pjob := range pchan {
			if _, ok := counted[pjob.path]; ok {
				continue
			}

			counted[pjob.path] = struct{}{}
			if pjob.err != nil {
				util.Warnf("\rCould not parse PDB file '%s' because: %s\n",
					pjob.path, pjob.err)
				errCnt++
			} else {
				successCnt++
			}
			util.Verbosef("\r%d of %d PDB files processed. "+
				"(%0.2f%% done with %d errors.)",
				successCnt+errCnt, total,
				100.0*(float64(successCnt+errCnt)/float64(total)),
				errCnt)
		}
		done <- struct{}{}
	}()
	return pchan, done
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
