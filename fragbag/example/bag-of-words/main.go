// Example bag-of-words shows how to compute a bag-of-words vector given a
// fragment library and a PDB file.
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

	// Initialize the fragment library whatever is provided. If the library
	// isn't valid or doesn't exist, exit with an error.
	lib, err := fragbag.NewLibrary(flag.Arg(0))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("Using library %s.\n", lib)

	for _, pdbfile := range flag.Args()[1:] {
		entry, err := pdb.New(pdbfile)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("Computing the bag-of-words vector for %s.\n", entry.Name())
		fmt.Println(lib.NewBowPDB(entry))
		fmt.Println("----------------------------------------------")
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
