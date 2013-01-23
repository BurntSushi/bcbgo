package main

import (
	"encoding/gob"
	"flag"
	"log"
	"os"
	"runtime"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/hhfrag"
)

var (
	hhfragConf = hhfrag.DefaultConfig
	flagSeqDB  = "nr20"
	flagPdbDB  = "pdb-select25"
	flagCpu    = runtime.NumCPU()

	seqDB hhsuite.Database
	pdbDB hhfrag.PDBDatabase
)

func init() {
	log.SetFlags(0)

	flag.BoolVar(&hhfragConf.Blits, "blits", hhfragConf.Blits,
		"When set, hhblits will be used in lieu of hhsearch.")
	flag.IntVar(&hhfragConf.WindowMin, "win-min", hhfragConf.WindowMin,
		"The minimum HMM window size for HHfrag.")
	flag.IntVar(&hhfragConf.WindowMax, "win-max", hhfragConf.WindowMax,
		"The maximum HMM window size for HHfrag.")
	flag.IntVar(&hhfragConf.WindowIncrement, "win-inc",
		hhfragConf.WindowIncrement,
		"The sliding window increment for HHfrag.")
	flag.StringVar(&flagSeqDB, "seqdb", flagSeqDB,
		"The sequence database used to generate the query HHM.")
	flag.StringVar(&flagPdbDB, "pdbdb", flagPdbDB,
		"The PDB/HHM database used to assign fragments.")
	flag.IntVar(&flagCpu, "cpu", flagCpu,
		"The max number of CPUs to use.")

	flag.Usage = usage
	flag.Parse()

	seqDB = hhsuite.Database(flagSeqDB)
	pdbDB = hhfrag.PDBDatabase(flagPdbDB)

	runtime.GOMAXPROCS(flagCpu)
}

func usage() {
	log.Printf("Usage: hhfrag-map [flags] target-fasta out-fmap\n")
	flag.PrintDefaults()
}

func main() {
	if flag.NArg() != 2 {
		usage()
	}

	fasInp := flag.Arg(0)
	fmapOut := flag.Arg(1)

	fmap, err := hhfragConf.MapFromFasta(pdbDB, seqDB, fasInp)
	assert(err)

	out, err := os.Create(fmapOut)
	assert(err)
	w := gob.NewEncoder(out)
	err = w.Encode(fmap)
	assert(err)
}

func assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
