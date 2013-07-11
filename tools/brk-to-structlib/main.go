package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/TuftsBCB/io/pdb"
)

var flagOverwrite = false

func init() {
	flag.BoolVar(&flagOverwrite, "overwrite", flagOverwrite,
		"When set, any existing database will be completely overwritten.")

	util.FlagParse("kolodny-brk-file struct-lib-outfile", "")
	util.AssertLeastNArg(2)
}

func main() {
	brkFile := util.Arg(0)
	saveto := util.Arg(1)

	util.AssertOverwritable(saveto, flagOverwrite)

	fbrk := util.OpenFile(brkFile)
	defer fbrk.Close()

	brkContents, err := ioutil.ReadAll(fbrk)
	util.Assert(err, "Could not read '%s'", brkFile)

	lib := fragbag.NewStructureLibrary(path.Base(brkFile))

	fragments := bytes.Split(brkContents, []byte("TER"))
	for i, fragment := range fragments {
		fragment = bytes.TrimSpace(fragment)
		if len(fragment) == 0 {
			continue
		}
		atoms := coords(i, fragment)
		lib.Add(atoms)
	}

	savetof := util.CreateFile(saveto)
	defer savetof.Close()

	lib.Save(savetof)
}

func coords(num int, atomRecords []byte) []pdb.Coords {
	r := bytes.NewReader(atomRecords)
	name := fmt.Sprintf("fragment %d", num)

	entry, err := pdb.Read(r, name)
	util.Assert(err, "Fragment contents could not be read in PDB format")

	atoms := entry.OneChain().CaAtoms()
	if len(atoms) == 0 {
		util.Fatalf("Fragment %d has no ATOM coordinates.", num)
	}
	return atoms
}
