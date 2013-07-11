package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/BurntSushi/bcbgo/bow"
	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/TuftsBCB/io/pdb"
)

var (
	flagFragbag  string
	flagOldStyle bool
)

func init() {
	flag.StringVar(&flagFragbag, "fragbag", "fragbag",
		"The old fragbag executable.")
	flag.BoolVar(&flagOldStyle, "oldstyle", false,
		"When true, NewBowPDBOldStyle will be used to compute BOW vectors.")

	util.FlagParse(
		"library-file pdb-file [pdb-file ...]",
		"Note that if the old library and the new library don't have the\n"+
			"same number of fragments and the same fragment size, bad things\n"+
			"will happen.\n")
	util.AssertLeastNArg(2)
}

func stderrf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

func main() {
	libFile := util.Arg(0)
	lib := util.FragmentLibrary(libFile)

	stderrf("Loading PDB files into memory...\n")
	entries := make([]*pdb.Entry, util.NArg()-1)
	for i, pdbfile := range flag.Args()[1:] {
		entries[i] = util.PDBRead(pdbfile)
	}

	stderrf("Comparing the results of old fragbag and new fragbag on " +
		"each PDB file...\n")
	for _, entry := range entries {
		stderrf("Testing %s...\n", entry.Path)
		fmt.Printf("Testing %s\n", entry.Path)

		// Try to run old fragbag first. The output is an old-style BOW.
		oldBowStr, err := runOldFragbag(libFile, entry.Path, lib.Size(),
			lib.FragmentSize())
		if err != nil {
			fmt.Println(err)
			fmt.Printf("The output was:\n%s\n", oldBowStr)
			divider()
			continue
		}

		oldBow, err := bow.NewOldStyleBow(lib.Size(), oldBowStr)
		if err != nil {
			fmt.Printf("Could not parse the following as an old style "+
				"BOW:\n%s\n", oldBowStr)
			fmt.Printf("%s\n", err)
			divider()
			continue
		}

		// Now use package fragbag to compute a BOW.
		var newBow bow.BOW
		if flagOldStyle {
			newBow = bow.ComputeBOW(lib, bow.PDBEntryOldStyle{entry})
		} else {
			newBow = bow.ComputeBOW(lib, entry)
		}

		// Create a diff and check if they are the same. If so, we passed.
		// Otherwise, print an error report.
		diff := bow.NewBowDiff(oldBow, newBow)
		if diff.IsSame() {
			fmt.Println("PASSED.")
			divider()
			continue
		}

		// Ruh roh...
		fmt.Println("FAILED.")
		fmt.Printf("\nOld BOW:\n%s\n\nNew BOW:\n%s\n", oldBow, newBow)
		fmt.Printf("\nDiff:\n%s\n", diff)
		divider()
	}
	stderrf("Done!\n")
}

func runOldFragbag(libFile, pdbFile string, size, fraglen int) (string, error) {
	cmd := []string{
		flagFragbag,
		"-l", libFile,
		fmt.Sprintf("%d", size),
		"-z", fmt.Sprintf("%d", fraglen),
		"-p", pdbFile,
		"-c"}
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)),
			fmt.Errorf("There was an error executing: %s\n%s",
				strings.Join(cmd, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func divider() {
	fmt.Println("----------------------------------------------------")
}
