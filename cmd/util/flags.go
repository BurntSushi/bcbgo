package util

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/hhfrag"
)

var (
	FlagCpu = runtime.NumCPU()

	FlagPdbDir = "/data/bio/pdb"

	flagSeqDB = "nr20"
	FlagSeqDB hhsuite.Database

	flagPdbHhmDB = "pdb-select25-2012"
	FlagPdbHhmDB hhfrag.PDBDatabase

	FlagBlits = false
)

func init() {
	log.SetFlags(0)
}

type commonFlag struct {
	set, init func()
	use       bool
}

var commonFlags = map[string]*commonFlag{
	"cpu": {
		set: func() {
			flag.IntVar(&FlagCpu, "cpu", FlagCpu,
				"The max number of CPUs to use.")
		},
		init: func() {
			runtime.GOMAXPROCS(FlagCpu)
		},
	},
	"pdb-dir": {
		set: func() {
			flag.StringVar(&FlagPdbDir, "pdb-dir", FlagPdbDir,
				"The path to the directory containing the PDB database.")
		},
	},
	"seq-db": {
		set: func() {
			flag.StringVar(&flagSeqDB, "seq-db", flagSeqDB,
				"The sequence database used to generate the query HHM.")
		},
		init: func() {
			FlagSeqDB = hhsuite.Database(flagSeqDB)
		},
	},
	"pdb-hhm-db": {
		set: func() {
			flag.StringVar(&flagPdbHhmDB, "pdb-hhm-db", flagPdbHhmDB,
				"The PDB/HHM database used to assign fragments.")
		},
		init: func() {
			FlagPdbHhmDB = hhfrag.PDBDatabase(flagPdbHhmDB)
		},
	},
	"blits": {
		set: func() {
			flag.BoolVar(&FlagBlits, "blits", FlagBlits,
				"When set, hhblits will be used in lieu of hhsearch.")
		},
	},
}

func FlagUse(names ...string) {
	for _, name := range names {
		commonFlags[name].use = true
	}
}

// Usage just calls `flag.Usage`. It's included here to avoid
// an extra import to `flag` just to call Usage.
func Usage() {
	flag.Usage()
}

// Arg just calls `flag.Arg`. It's included here to avoid
// an extra import to `flag` just to call Arg.
func Arg(i int) string {
	return flag.Arg(i)
}

// NArg just calls `flag.NArg`. It's included here to avoid
// an extra import to `flag` just to call NArg.
func NArg() int {
	return flag.NArg()
}

func FlagParse(positional string, desc string) {
	for _, fl := range commonFlags {
		if fl.use {
			fl.set()
		}
	}

	flag.Usage = func() {
		log.Printf("Usage: %s [flags] %s\n\n",
			path.Base(os.Args[0]), positional)
		if len(desc) > 0 {
			log.Printf("%s\n", desc)
		}
		flag.VisitAll(func(fl *flag.Flag) {
			var def string
			if len(fl.DefValue) > 0 {
				def = fmt.Sprintf(" (default: %s)", fl.DefValue)
			}

			usage := strings.Replace(fl.Usage, "\n", "\n    ", -1)
			log.Printf("-%s%s\n", fl.Name, def)
			log.Printf("    %s\n", usage)
		})
		os.Exit(1)
	}
	flag.Parse()

	for _, fl := range commonFlags {
		if fl.use && fl.init != nil {
			fl.init()
		}
	}
}
