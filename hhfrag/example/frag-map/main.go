package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/hhfrag"
)

var (
	flagSeqDB      = "nr20"
	flagPdbDB      = "pdb-select25"
	flagHHM        = ""
	flagBlits      = false
	flagCpuProfile = ""

	seqDB hhsuite.Database
	pdbDB hhfrag.PDBDatabase
)

func init() {
	log.SetFlags(0)

	flag.StringVar(&flagSeqDB, "seqdb", flagSeqDB,
		"The sequence database to use to generate an HHM.")
	flag.StringVar(&flagPdbDB, "pdbdb", flagPdbDB,
		"The PDB/HHM database to use to classify fragments.")
	flag.StringVar(&flagHHM, "hhm", flagHHM,
		"An HHM file to use. (Skips HHM generation of query.)")
	flag.BoolVar(&flagBlits, "blits", flagBlits,
		"If set, hhblits will be used in liu of hhsearch.")
	flag.StringVar(&flagCpuProfile, "cpuprofile", flagCpuProfile,
		"When set, a CPU profile will be written to the file specified.")

	flag.Parse()

	seqDB = hhsuite.Database(flagSeqDB)
	pdbDB = hhfrag.PDBDatabase(flagPdbDB)

	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	if len(flagCpuProfile) > 0 {
		f, err := os.Create(flagCpuProfile)
		if err != nil {
			log.Fatalln(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if flag.NArg() != 1 {
		log.Fatalln("No input file specified.")
	}

	var fmap hhfrag.FragmentMap
	var err error
	conf := hhfrag.DefaultConfig
	conf.Blits = flagBlits
	if len(flagHHM) > 0 {
		fmap, err = conf.MapFromHHM(pdbDB, seqDB, flag.Arg(0), flagHHM)
	} else {
		fmap, err = conf.MapFromFasta(pdbDB, seqDB, flag.Arg(0))
	}
	assert(err)

	for _, frags := range fmap {
		fmt.Printf("\nSEGMENT: %d %d (%d)\n",
			frags.Start, frags.End, len(frags.Frags))
		frags.Write(os.Stdout)
	}
}

func assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
