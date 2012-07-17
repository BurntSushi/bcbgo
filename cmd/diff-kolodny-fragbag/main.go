package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/pdb"
)

var (
	flagFragbag  string
	flagOldStyle bool
)

func main() {
	if flag.NArg() < 3 {
		usage()
	}

	oldLibFile, newLibPath := flag.Arg(0), flag.Arg(1)
	lib, err := fragbag.NewLibrary(newLibPath)
	if err != nil {
		fatalf("%s\n", err)
	}

	errorf("Loading PDB files into memory...\n")
	entries := make([]*pdb.Entry, flag.NArg()-2)
	for i, pdbfile := range flag.Args()[2:] {
		entries[i], err = pdb.New(pdbfile)
		if err != nil {
			fatalf("%s\n", err)
		}
	}

	errorf("Comparing the results of old fragbag and new fragbag on each " +
		"PDB file...\n")
	for _, entry := range entries {
		errorf("Testing %s...\n", entry.Name())
		fmt.Printf("Testing %s\n", entry.Name())

		// Try to run old fragbag first. The output is an old-style BOW.
		oldBowStr, err := runOldFragbag(oldLibFile, entry.Path, lib.Size(),
			lib.FragmentSize())
		if err != nil {
			fmt.Println(err)
			fmt.Printf("The output was:\n%s\n", oldBowStr)
			divider()
			continue
		}

		oldBow, err := lib.NewOldStyleBow(oldBowStr)
		if err != nil {
			fmt.Printf("Could not parse the following as an old style "+
				"BOW:\n%s\n", oldBowStr)
			fmt.Printf("%s\n", err)
			divider()
			continue
		}

		// Now use package fragbag to compute a BOW.
		var newBow fragbag.BOW
		if flagOldStyle {
			newBow = lib.NewBowPDBOldStyle(entry)
		} else {
			newBow = lib.NewBowPDB(entry)
		}

		// Create a diff and check if they are the same. If so, we passed.
		// Otherwise, print an error report.
		diff := fragbag.NewBowDiff(oldBow, newBow)
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
	errorf("Done!\n")
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

func errorf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

func fatalf(format string, v ...interface{}) {
	errorf(format, v...)
	os.Exit(1)
}

func init() {
	flag.StringVar(&flagFragbag, "fragbag", "fragbag",
		"The old fragbag executable.")
	flag.BoolVar(&flagOldStyle, "oldstyle", false,
		"When true, NewBowPDBOldStyle will be used to compute BOW vectors.")
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s [flags] "+
			"old-library-file new-library-path pdb-file [ pdb-file ... ]\n",
		path.Base(os.Args[0]))
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nNote that if the old library and the new "+
		"library don't have the same number of fragments and the same "+
		"fragment size, bad things will happen.\n")
	os.Exit(1)
}
