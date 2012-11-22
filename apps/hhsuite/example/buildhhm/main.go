// Example buildhhm shows how to construct an HHM using HHblits and HHmake
// from a single sequence FASTA file.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/io/hhm"
)

var (
	flagDatabase = "nr20"
	flagQuiet = false
)

func init() {
	log.SetFlags(0)

	flag.StringVar(&flagDatabase, "db", flagDatabase,
		"The database to use to generate MSAs.")
	flag.BoolVar(&flagQuiet, "quiet", flagQuiet,
		"When set, hhblits/hhmake output will be hidden.")
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	log.Println("Usage: buildhhm [flags] in-fasta-file out-hhm-file")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	if flag.NArg() != 2 {
		usage()
	}
	inFasta := flag.Arg(0)
	outHHM := flag.Arg(1)

	hhblits := hhsuite.HHBlitsDefault
	hhmake := hhsuite.HHMakePseudo
	hhblits.Verbose = !flagQuiet
	hhmake.Verbose = !flagQuiet

	HHM, err := hhsuite.BuildHHM(
		hhblits, hhmake, hhsuite.Database(flagDatabase), inFasta)
	if err != nil {
		log.Fatalf("Error building HHM: %s\n", err)
	}

	foutHHM, err := os.Create(outHHM)
	if err != nil {
		log.Fatalf("Error creating HHM file (%s): %s", outHHM, err)
	}
	if err := hhm.Write(foutHHM, HHM); err != nil {
		log.Fatalf("Error writing HHM (%s): %s", outHHM, err)
	}
}
