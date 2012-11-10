// Example bag-of-words shows how to compute a bag-of-words vector given a
// fragment library and a PDB file.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/pdb"
)

var (
	flagCpuProfile = ""
	flagGoMaxProcs = runtime.NumCPU()
	flagOverwrite  = false
	flagQuiet      = false
)

func writer(db *bowdb.DB,
	pchan chan progressJob, pool pool) (chan struct{}, error) {

	done := make(chan struct{}, 0)
	go func() {
		for result := range pool.results {
			pchan <- progressJob{result.chain.Entry.Path, nil}
			if err := db.Write(result.chain, result.bow); err != nil {
				fatalf("%s\n", err)
			}
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
				errorf("\rCould not parse PDB file '%s' because: %s\n",
					pjob.path, pjob.err)
				errCnt++
			} else {
				successCnt++
			}
			verbosef("\r%d of %d PDB files processed. "+
				"(%0.2f%% done with %d errors.)",
				successCnt+errCnt, total,
				100.0*(float64(successCnt+errCnt)/float64(total)),
				errCnt)
		}
		done <- struct{}{}
	}()
	return pchan, done
}

func main() {
	if flag.NArg() < 3 {
		usage()
	}
	dbPath := flag.Arg(0)
	libPath := flag.Arg(1)
	pdbFiles := flag.Args()[2:]

	if flagOverwrite {
		if err := os.RemoveAll(dbPath); err != nil {
			fatalf("Could not remove '%s' directory because: %s.", dbPath, err)
		}
	}
	if len(pdbFiles) == 1 && isDir(pdbFiles[0]) {
		pdbFiles = recursiveFilesInDir(pdbFiles[0])
	}

	lib, err := fragbag.NewLibrary(libPath)
	if err != nil {
		fatalf("Could not open fragment library '%s': %s\n", lib, err)
	}
	verbosef("Using library %s.\n", lib)

	db, err := bowdb.Create(lib, dbPath)
	if err != nil {
		fatalf("%s\n", err)
	}

	pool := newBowWorkers(lib, max(1, flagGoMaxProcs))
	progressChan, doneProgress := progress(len(pdbFiles))
	doneWriting, err := writer(db, progressChan, pool)
	if err != nil {
		fatalf("Could not create database file: %s\n", err)
	}

	if len(flagCpuProfile) > 0 {
		f, err := os.Create(flagCpuProfile)
		if err != nil {
			fatalf("%s\n", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	for _, pdbFile := range pdbFiles {
		entry, err := pdb.New(pdbFile)
		if err != nil {
			progressChan <- progressJob{pdbFile, err}
			continue
		}

		// If there aren't any protein chains, skip this entry but update
		// the progress bar.
		hasProteinChains := false
		for _, chain := range entry.Chains {
			if chain.ValidProtein() {
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
	verbosef("\n")

	if err := db.WriteClose(); err != nil {
		fatalf("There was an error closing the database: %s\n", err)
	}
}

func init() {
	flag.StringVar(&flagCpuProfile, "cpuprofile", flagCpuProfile,
		"When set, a CPU profile will be written to the file provided.")
	flag.IntVar(&flagGoMaxProcs, "p", flagGoMaxProcs,
		"The maximum number of CPUs that can be executing simultaneously.")
	flag.BoolVar(&flagOverwrite, "overwrite", flagOverwrite,
		"When set, any existing database will be completely overwritten.")
	flag.BoolVar(&flagQuiet, "quiet", flagQuiet,
		"When set, no progress bar will be shown.\n"+
			"\tErrors will still be printed to stderr.")
	flag.Usage = usage
	flag.Parse()

	runtime.GOMAXPROCS(flagGoMaxProcs)
}

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s database-path frag-lib-directory "+
			"(pdb-dir | (pdb-file [pdb-file ...]))\n",
		path.Base(os.Args[0]))
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr,
		"\nex. './%s data/fraglibs/centers400_11 data/samples/*.pdb'\n",
		path.Base(os.Args[0]))
	os.Exit(1)
}

func verbosef(format string, v ...interface{}) {
	if flagQuiet {
		return
	}
	fmt.Fprintf(os.Stderr, format, v...)
}

func errorf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

func fatalf(format string, v ...interface{}) {
	errorf(format, v...)
	os.Exit(1)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isDir(f string) bool {
	fi, err := os.Stat(f)
	return err == nil && fi.IsDir()
}

func recursiveFilesInDir(dir string) []string {
	files := make([]string, 0, 50)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			errorf("Could not read '%s' because: %s\n", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}
