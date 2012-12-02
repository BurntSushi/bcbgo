package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/bcbgo/io/pdb"
)

var (
	flagPdbDir    = "/data/bio/pdb"
	flagSkipCheck = false
)

func init() {
	log.SetFlags(0)

	flag.StringVar(&flagPdbDir, "pdb-dir", flagPdbDir,
		"The path to the directory containing the PDB database.")
	flag.BoolVar(&flagSkipCheck, "skip-check", flagSkipCheck,
		"When set, PDB files will not be checked for irreparable corruption.")

	flag.Usage = usage
	flag.Parse()
}

func usage() {
	log.Println("Usage: gather-pdb-chains [flags] input-file output-dir")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	if flag.NArg() != 2 {
		usage()
	}

	outDir := flag.Arg(1)
	info, err := os.Stat(outDir)
	if err != nil {
		log.Fatalf("Directory '%s' is not accessible: %s", outDir, err)
	}
	if !info.IsDir() {
		log.Fatalf("'%s' is not a directory.", outDir)
	}

	f, err := os.Open(flag.Arg(0))
	assert(err)

	entries, err := getEntries(f)
	assert(err)

	// We have to traverse each PDB structure and make sure none of them
	// are corrupt. If they are, the user will have to manually collect them,
	// and probably fix the PDB entries themselves.
	if !flagSkipCheck {
		for _, entry := range entries {
			e, err := pdb.ReadPDB(entry.Path())
			if err != nil {
				log.Println(err)
				continue
			}

			chain := e.Chain(entry.Chain)
			if chain == nil {
				log.Fatalf("Could not find chain '%c' in PDB entry '%s'.",
					entry.Chain, entry.Path())
			}
			chain.SequenceCaAtoms()
		}
	}
	for _, entry := range entries {
		fname := fmt.Sprintf("%s%c.ent.gz", entry.IdCode, entry.Chain)
		dest := path.Join(outDir, fname)
		assert(copyFile(entry.Path(), dest))
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
	return path.Join(flagPdbDir, e.IdCode[1:3], fname)
}

func getEntries(f *os.File) ([]entry, error) {
	lines, err := readLines(f)
	assert(err)
	if len(lines) == 0 {
		log.Fatalf("Could not find any PDB entries in '%s'.", f.Name())
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

func readLines(f *os.File) ([]string, error) {
	buf := bufio.NewReader(f)
	lines := make([]string, 0, 100)
	for {
		line, err := buf.ReadString('\n')
		if err == io.EOF {
			if len(line) == 0 {
				break
			}
		} else if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func copyFile(src, dest string) error {
	fsrc, err := os.Open(src)
	if err != nil {
		return err
	}

	fdest, err := os.Create(dest)
	if err != nil {
		return err
	}

	_, err = io.Copy(fdest, fsrc)
	return err
}

func assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
