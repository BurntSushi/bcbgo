package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/hhfrag"
	"github.com/BurntSushi/bcbgo/io/fasta"
	"github.com/BurntSushi/bcbgo/io/hhm"
	"github.com/BurntSushi/bcbgo/seq"
)

var (
	flagFasta = ""
	flagHHM   = ""
	flagSeqDB = "nr20"
	flagPdbDB = "pdb-select25"
	flagStart = -1
	flagEnd   = -1

	seqDB hhsuite.Database
	pdbDB hhfrag.PDBDatabase
)

func init() {
	log.SetFlags(0)

	flag.StringVar(&flagFasta, "fasta", flagFasta,
		"An input FASTA file from which to build an HHM.")
	flag.StringVar(&flagHHM, "hhm", flagHHM,
		"An input HHM file; skips building HHM.")
	flag.StringVar(&flagSeqDB, "seqdb", flagSeqDB,
		"The sequence database to use to generate an HHM.")
	flag.StringVar(&flagPdbDB, "pdbdb", flagPdbDB,
		"The PDB/HHM database to use to classify fragments.")
	flag.IntVar(&flagStart, "start", flagStart,
		"The start location of the query fragment window to classify.")
	flag.IntVar(&flagEnd, "end", flagEnd,
		"The end location of the query fragment window to classify.")

	flag.Parse()

	seqDB = hhsuite.Database(flagSeqDB)
	pdbDB = hhfrag.PDBDatabase(flagPdbDB)
}

func main() {
	fs, fe := getFragmentWindow()
	qseq, qhhm := getQueryHHM()

	part := qhhm.Slice(fs, fe)

	frags, err := hhfrag.FindFragments(pdbDB, part, qseq, false)
	assert(err)

	tabw := tabwriter.NewWriter(os.Stderr, 0, 4, 4, ' ', 0)
	fmt.Fprintln(tabw, "Hit\tQuery\tTemplate\tProb\tCorrupt")
	for _, frag := range frags {
		var corruptStr string
		if frag.IsCorrupt() {
			corruptStr = "\tcorrupt"
		}
		fmt.Fprintf(tabw, "%s\t(%d-%d)\t(%d-%d)\t%f%s\n",
			frag.Template.Name,
			frag.Hit.QueryStart, frag.Hit.QueryEnd,
			frag.Hit.TemplateStart, frag.Hit.TemplateEnd,
			frag.Hit.Prob,
			corruptStr)
	}
	tabw.Flush()
}

func getFragmentWindow() (int, int) {
	if flagStart == -1 {
		log.Fatalln("Please set the '--start' flag.")
	}
	if flagEnd == -1 {
		log.Fatalln("Please set the '--end' flag.")
	}
	return flagStart - 1, flagEnd
}

func getQueryHHM() (seq.Sequence, *hhm.HHM) {
	if len(flagFasta) == 0 {
		log.Fatalln("Please set the '--fasta' flag.")
	}

	seqs, err := fasta.NewReader(openFile(flagFasta)).ReadAll()
	assert(err)
	qseq := seqs[0]

	if len(flagHHM) > 0 {
		queryHHM, err := hhm.Read(openFile(flagHHM))
		assert(err)
		return qseq, queryHHM
	}

	queryHHM, err := hhsuite.BuildHHM(
		hhsuite.HHBlitsDefault, hhsuite.HHMakePseudo, seqDB, flagFasta)
	assert(err)
	return qseq, queryHHM
}

func assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func openFile(fpath string) *os.File {
	f, err := os.Open(fpath)
	if err != nil {
		log.Fatalln(err)
	}
	return f
}
