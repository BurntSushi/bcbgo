// Example rmsd shows how to compute the RMSD between any two sets of atoms
// from PDB files. The sets must be of equal length.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/BurntSushi/bcbgo/pdb"
	"github.com/BurntSushi/bcbgo/rmsd"
)

func main() {
	if flag.NArg() != 8 {
		usage()
	}

	pdbf1, c1, s1, e1 := flag.Arg(0), flag.Arg(1), flag.Arg(2), flag.Arg(3)
	pdbf2, c2, s2, e2 := flag.Arg(4), flag.Arg(5), flag.Arg(6), flag.Arg(7)

	// Build pdb.Entry values. If anything goes wrong, quit!
	entry1, err := pdb.New(pdbf1)
	if err != nil {
		fmt.Println(err)
		return
	}
	entry2, err := pdb.New(pdbf2)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Make sure the chains specified exist!
	chain1, ok := entry1.Chains[c1[0]]
	if !ok {
		fmt.Printf("The chain '%s' could not be found in '%s'.\n", c1, pdbf1)
		os.Exit(1)
	}
	chain2, ok := entry2.Chains[c2[0]]
	if !ok {
		fmt.Printf("The chain '%s' could not be found in '%s'.\n", c2, pdbf2)
		os.Exit(1)
	}

	// Now make sure the slice numbers are valid integers.
	s1n, e1n, s2n, e2n := parseInt(s1), parseInt(e1), parseInt(s2), parseInt(e2)

	// Now we need to traverse the atoms for each chain, and pick out the
	// carbon-alpha atoms corresponding to the specified residue range.
	struct1, struct2 := make(pdb.Atoms, 0), make(pdb.Atoms, 0)
	for _, atom := range chain1.CaAtoms {
		if atom.ResidueInd >= s1n && atom.ResidueInd <= e1n {
			struct1 = append(struct1, atom)
		}
	}
	for _, atom := range chain2.CaAtoms {
		if atom.ResidueInd >= s2n && atom.ResidueInd <= e2n {
			struct2 = append(struct2, atom)
		}
	}

	// Verify that there are some atoms to compute an RMSD with.
	if len(struct1) == 0 {
		fmt.Printf("The range %d-%d does not correspond to any carbon-alpha "+
			"atoms.\n", s1n, e1n)
		os.Exit(1)
	}
	if len(struct2) == 0 {
		fmt.Printf("The range %d-%d does not correspond to any carbon-alpha "+
			"atoms.\n", s2n, e2n)
		os.Exit(1)
	}

	// Verify that the ranges are the same length.
	if len(struct1) != len(struct2) {
		fmt.Printf("The range %d-%d corresponds to %d carbon-alpha atoms "+
			"while the range %d-%d corresponds to %d carbon-alpha atoms. "+
			"Both ranges must correspond to the same number of "+
			"carbon-alpha atoms.\n",
			s1n, e1n, len(struct1), s2n, e2n, len(struct2))
		os.Exit(1)
	}

	// Now compute the RMSD of the corresponding atom sets.
	fmt.Println(rmsd.RMSD(struct1, struct2))
}

func parseInt(numStr string) int {
	num, err := strconv.ParseInt(numStr, 10, 32)
	if err != nil {
		fmt.Printf("Could not parse '%s' as an integer.\n", numStr)
		os.Exit(1)
	}
	return int(num)
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
		"\nex. './%s "+
			"../../../data/samples/sample1.pdb A 1 10 "+
			"../../../data/samples/sample1.pdb A 10 20'\n",
		path.Base(os.Args[0]))
	os.Exit(1)
}
