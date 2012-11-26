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
	flagBlits = false
	flagSeqDB = "nr20"
	flagPdbDB = "pdb-select25"
	flagCpu   = runtime.GOMAXPROCS(0)

	seqDB hhsuite.Database
	pdbDB hhfrag.PDBDatabase
)

func init() {
	log.SetFlags(0)

	flag.BoolVar(&flagBlits, "blits", flagBlits,
		"When set, hhblits will be used in lieu of hhsearch.")
	flag.StringVar(&flagSeqDB, "seqdb", flagSeqDB,
		"The sequence database used to generate the query HHM.")
	flag.StringVar(&flagPdbDB, "pdbdb", flagPdbDB,
		"The PDB/HHM database used to assignment fragments.")
	flag.IntVar(&flagCpu, "cpu", flagCpu,
		"The max number of CPUs to use.")

	flag.Usage = usage
	flag.Parse()

	seqDB = hhsuite.Database(flagSeqDB)
	pdbDB = hhfrag.PDBDatabase(flagPdbDB)

	runtime.GOMAXPROCS(flagCpu)
}

func usage() {
	log.Printf("Usage: hhfrag-stats [flags] target-fasta\n")
	flag.PrintDefaults()
}

func main() {
	if flag.NArg() != 1 {
		log.Println("One input file required.\n")
		usage()
	}

	fasInp := flag.Arg(0)
	conf := hhfrag.DefaultConfig
	conf.Blits = flagBlits
	fmap, err := conf.MapFromFasta(pdbDB, seqDB, fasInp)
	assert(err)

	w := gob.NewEncoder(os.Stdout)
	err = w.Encode(fmap)
	assert(err)
}

func assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
