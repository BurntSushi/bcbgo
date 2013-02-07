package util

import (
	"encoding/gob"
	"io"
	"os"
	"strconv"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/hhfrag"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

func FragmentLibrary(path string) *fragbag.Library {
	lib, err := fragbag.NewLibrary(path)
	Assert(err, "Could not open fragment library '%s'", path)
	return lib
}

func PDBRead(path string) *pdb.Entry {
	entry, err := pdb.ReadPDB(path)
	Assert(err, "Could not open PDB file '%s'", path)
	return entry
}

func FmapRead(path string) hhfrag.FragmentMap {
	var fmap hhfrag.FragmentMap
	r := gob.NewDecoder(OpenFile(path))
	Assert(r.Decode(&fmap), "Could not GOB decode fragment map '%s'", path)
	return fmap
}

func FmapWrite(w io.Writer, fmap hhfrag.FragmentMap) {
	encoder := gob.NewEncoder(w)
	Assert(encoder.Encode(fmap), "Could not GOB encode fragment map")
}

func BOWRead(path string) fragbag.BOW {
	var bow fragbag.BOW
	r := gob.NewDecoder(OpenFile(path))
	Assert(r.Decode(&bow), "Could not GOB decode BOW '%s'", path)
	return bow
}

func BOWWrite(w io.Writer, bow fragbag.BOW) {
	encoder := gob.NewEncoder(w)
	Assert(encoder.Encode(bow), "Could not GOB encode BOW")
}

func OpenFile(path string) *os.File {
	f, err := os.Open(path)
	Assert(err, "Could not open file '%s'", path)
	return f
}

func CreateFile(path string) *os.File {
	f, err := os.Create(path)
	Assert(err, "Could not create file '%s'", path)
	return f
}

func ParseInt(str string) int {
	num, err := strconv.ParseInt(str, 10, 32)
	Assert(err, "Could not parse '%s' as an integer", str)
	return int(num)
}
