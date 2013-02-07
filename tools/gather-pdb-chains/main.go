package main

// This needs to be updated to read straight from PDB select 25 databases.
// Or at least, be capable of it (we may want to read lists of PDB
// entries from other kinds of sources).

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

var (
	flagSkipCheck = false
)

func init() {
	flag.BoolVar(&flagSkipCheck, "skip-check", flagSkipCheck,
		"When set, PDB files will not be checked for irreparable corruption.")

	util.FlagUse("pdb-dir")
	util.FlagParse("input-file output-dir", "")
	util.AssertNArg(2)
}

func main() {
	outDir := util.Arg(1)
	util.AssertIsDir(outDir)

	entries, err := getEntries(util.OpenFile(flag.Arg(0)))
	util.Assert(err)

	// We have to traverse each PDB structure and make sure none of them
	// are corrupt. If they are, the user will have to manually collect them,
	// and probably fix the PDB entries themselves.
	if !flagSkipCheck {
		for _, entry := range entries {
			e, err := pdb.ReadPDB(entry.Path())
			if util.Warning(err) {
				continue
			}

			chain := e.Chain(entry.Chain)
			if chain == nil {
				util.Warnf("Could not find chain '%c' in PDB entry '%s'.",
					entry.Chain, entry.Path())
			}
			chain.SequenceCaAtoms()
		}
	}
	for _, entry := range entries {
		fname := fmt.Sprintf("%s%c.ent.gz", entry.IdCode, entry.Chain)
		util.CopyFile(entry.Path(), path.Join(outDir, fname))
	}
}

type entry struct {
	IdCode string
	Chain  byte
}

func newEntry(idchain string) entry {
	return entry{strings.ToLower(idchain[0:4]), idchain[4]}
}

func (e entry) Path() string {
	fname := fmt.Sprintf("pdb%s.ent.gz", e.IdCode)
	return path.Join(util.FlagPdbDir, e.IdCode[1:3], fname)
}

func getEntries(f *os.File) ([]entry, error) {
	lines := util.ReadLines(f)
	if len(lines) == 0 {
		util.Fatalf("Could not find any PDB entries in '%s'.", f.Name())
	}

	entries := make([]entry, 0, len(lines))

	// If the first line only has 5 characters, then we're dealing with a
	// regular format. (One pdb entry/chain on each line.)
	if len(lines[0]) == 5 {
		for _, line := range lines {
			entries = append(entries, newEntry(line[0:5]))
		}
	} else { // otherwise we have a PDB Select 25 database
		for i, line := range lines {
			if strings.HasPrefix(line, "#") {
				continue
			}
			fs := strings.Fields(line)
			if len(fs) < 2 {
				return nil, fmt.Errorf("Invalid PDB Select 25 database "+
					"format. Expected at least two whitespace delimited "+
					"fields on line %d, but found only %d.", i+1, len(fs))
			}
			if len(fs[1]) != 5 {
				return nil, fmt.Errorf("Expected a PDB identifier/chain on "+
					"line %d, but found '%s' instead.", i+1, fs[1])
			}
			entries = append(entries, newEntry(fs[1][0:5]))
		}
	}
	return entries, nil
}
