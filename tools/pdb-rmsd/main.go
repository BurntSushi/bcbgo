package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/BurntSushi/bcbgo/io/pdb"
	"github.com/BurntSushi/bcbgo/rmsd"
)

func main() {
	if flag.NArg() != 8 {
		usage()
	}

	// Nice aliases.
	pdbf1, chain1, s1, e1 := flag.Arg(0), flag.Arg(1), flag.Arg(2), flag.Arg(3)
	pdbf2, chain2, s2, e2 := flag.Arg(4), flag.Arg(5), flag.Arg(6), flag.Arg(7)

	// Build pdb.Entry values. If anything goes wrong, quit!
	entry1, err := pdb.ReadPDB(pdbf1)
	if err != nil {
		fatalf("%s", err)
	}
	entry2, err := pdb.ReadPDB(pdbf2)
	if err != nil {
		fatalf("%s", err)
	}

	// Now make sure the slice numbers are valid integers.
	s1n, e1n, s2n, e2n := parseInt(s1), parseInt(e1), parseInt(s2), parseInt(e2)

	r, err := rmsd.PDB(entry1, chain1[0], s1n, e1n, entry2, chain2[0], s2n, e2n)
	if err != nil {
		fatalf("%s", err)
	}
	fmt.Println(r)
}

func parseInt(numStr string) int {
	num, err := strconv.ParseInt(numStr, 10, 32)
	if err != nil {
		fatalf("Could not parse '%s' as an integer.", numStr)
	}
	return int(num)
}

func fatalf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
	fmt.Fprintln(os.Stderr, "")
	os.Exit(1)
}

func init() {
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s pdb-file chain-id start stop pdb-file chain-id start stop\n",
		path.Base(os.Args[0]))
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr,
		"\nex. './%s sample1.pdb A 30 40 sample1.pdb A 40 50'\n",
		path.Base(os.Args[0]))
	os.Exit(1)
}
