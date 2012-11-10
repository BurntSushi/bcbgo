// Example rmsd shows how to use the matt package to invoke Matt on multiple
// argument sets in parallel.
package main

import (
	"fmt"
	"path"
	"strings"

	"github.com/BurntSushi/bcbgo/apps/matt"
)

func main() {
	// Build the argument sets to pass to Matt to run in parallel.
	// Other options like specifying the chain or residue ranges can be
	// tweaked by using the matt.PDBArg type directly.
	pdbArgs := [][]matt.PDBArg{
		{arg("sample1.pdb"), arg("sample2.pdb")},
		{arg("sample2.pdb"), arg("sample3.pdb")},
		{arg("sample1.pdb"), arg("sample3.pdb")},
		{arg("sample1.pdb"), arg("sample2.pdb"), arg("sample3.pdb")},
	}

	// Set 'Verbose' to false to not output anything.
	// (If there's an error, stderr will be logged to the error return.)
	config := matt.DefaultConfig
	config.Verbose = true

	// Run all of the argument sets with Matt in parallel. The indices in
	// both 'results' and 'errs' correspond to the indices in 'pdbArgs'.
	//
	// RunAll blocks until *all* alignments are finished.
	results, errs := config.RunAll(pdbArgs)

	// Loop through each error. If the error is nil, then there is result.
	// Otherwise, print the error.
	for i, err := range errs {
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("%s\n", argsetString(pdbArgs[i]))
			fmt.Printf("\tCore length: %d\n", results[i].CoreLength)
			fmt.Printf("\tRMSD: %0.4f\n", results[i].RMSD)
			fmt.Printf("\tP-value: %0.4f\n", results[i].Pval)
		}
	}
}

func arg(loc string) matt.PDBArg {
	return matt.PDBArg{
		Path: fmt.Sprintf("../../../../data/samples/%s",loc),
	}
}

func argsetString(argset []matt.PDBArg) string {
	basenames := make([]string, len(argset))
	for i, arg := range argset {
		basenames[i] = path.Base(arg.Path)
	}
	return fmt.Sprintf("(%s)", strings.Join(basenames, ", "))
}
