package main

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"sync"

	"github.com/BurntSushi/bcbgo/cmd/util"
)

func init() {
	util.FlagUse("cpu", "seq-db", "pdb-hhm-db", "blits",
		"hhfrag-min", "hhfrag-max", "hhfrag-inc")
	util.FlagParse("out-dir target-fasta", "")
	util.AssertLeastNArg(2)
}

func main() {
	outDir := util.Arg(0)
	fasInps := util.Args()[1:]

	util.Assert(os.MkdirAll(outDir, 0777))

	fastaChan := make(chan string)
	wg := new(sync.WaitGroup)
	for i := 0; i < max(1, runtime.GOMAXPROCS(0)); i++ {
		go func() {
			wg.Add(1)
			for fasta := range fastaChan {
				fmap := util.GetFmap(fasta)
				outF := path.Join(outDir, fmt.Sprintf("%s.fmap", fmap.Name))
				util.FmapWrite(util.CreateFile(outF), fmap)
			}
			wg.Done()
		}()
	}

	for _, fasta := range fasInps {
		fastaChan <- fasta
	}

	close(fastaChan)
	wg.Wait()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
