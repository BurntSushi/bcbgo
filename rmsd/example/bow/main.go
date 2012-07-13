// Example bow shows how to compute a bag-of-words Fragbag vector for
// PDB entries.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/pdb"
)

func main() {
	if flag.NArg() < 2 {
		usage()
	}

	// Look for and create a new library of fragments at the path provided.
	lib, err := fragbag.NewLibrary(flag.Arg(0))
	if err != nil {
		fmt.Println(err)
		return
	}

	// For each PDB file provided, compute the bag-of-words vector against
	// the provided library.
	for _, pdbfile := range flag.Args()[1:] {
		entry, err := pdb.New(pdbfile)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("> ", pdbfile)
		fmt.Println(lib.NewBowPDB(entry))
		fmt.Println("---------------------------------------------------")
	}
}

func init() {
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s frag-lib-directory pdb-file [ pdb-file ... ]\n",
		path.Base(os.Args[0]))
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr,
		"\nex. './%s ../../../data/fraglibs/centers400_11 "+
			"../../../data/samples/1ctf.pdb'\n",
		path.Base(os.Args[0]))
	os.Exit(1)
}
