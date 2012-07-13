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
	if flag.NArg() != 2 {
		usage()
	}

	lib, err := fragbag.NewLibrary(flag.Arg(0))
	if err != nil {
		fmt.Println(err)
		return
	}

	pdb, err := pdb.New(flag.Arg(1))
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(lib.NewBowPDB(pdb))
}

func init() {
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s frag-lib-directory pdb-file\n",
		path.Base(os.Args[0]))
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr,
		"\nex. './%s ../../../data/fraglibs/centers400_11 "+
			"../../../data/samples/1ctf.pdb'\n",
		path.Base(os.Args[0]))
	os.Exit(1)
}
