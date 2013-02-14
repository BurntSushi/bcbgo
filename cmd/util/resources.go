package util

import (
	"encoding/gob"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/bcbgo/bow"
	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/hhfrag"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

func FragmentLibrary(path string) *fragbag.Library {
	lib, err := fragbag.NewLibrary(path)
	Assert(err, "Could not open fragment library '%s'", path)
	return lib
}

func OpenBOWDB(path string) *bow.DB {
	db, err := bow.OpenDB(path)
	Assert(err, "Could not open BOW database '%s'", path)
	return db
}

func PDBRead(path string) *pdb.Entry {
	entry, err := pdb.ReadPDB(path)
	Assert(err, "Could not open PDB file '%s'", path)
	return entry
}

func GetFmap(fpath string) *hhfrag.FragmentMap {
	var fmap *hhfrag.FragmentMap
	var err error

	switch {
	case IsFasta(fpath):
		fmap, err = HHfragConf.MapFromFasta(FlagPdbHhmDB, FlagSeqDB, fpath)
		Assert(err, "Could not generate map from '%s'", fpath)
	case IsFmap(fpath):
		fmap = FmapRead(fpath)
	default:
		Fatalf("File '%s' is not a fasta or fmap file.", fpath)
	}

	return fmap
}

func FmapRead(path string) *hhfrag.FragmentMap {
	var fmap *hhfrag.FragmentMap
	f := OpenFile(path)
	defer f.Close()

	r := gob.NewDecoder(f)
	Assert(r.Decode(&fmap), "Could not GOB decode fragment map '%s'", path)
	return fmap
}

func FmapWrite(w io.Writer, fmap *hhfrag.FragmentMap) {
	encoder := gob.NewEncoder(w)
	Assert(encoder.Encode(fmap), "Could not GOB encode fragment map")
}

func BOWRead(path string) bow.BOW {
	var bow bow.BOW
	f := OpenFile(path)
	defer f.Close()

	r := gob.NewDecoder(f)
	Assert(r.Decode(&bow), "Could not GOB decode BOW '%s'", path)
	return bow
}

func BOWWrite(w io.Writer, bow bow.BOW) {
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

func IsFasta(fpath string) bool {
	suffix := func(ext string) bool {
		return strings.HasSuffix(fpath, ext)
	}
	return suffix(".fasta") || suffix(".fas")
}

func IsFmap(fpath string) bool {
	return strings.HasSuffix(fpath, ".fmap")
}

func IsPDB(fpath string) bool {
	suffix := func(ext string) bool {
		return strings.HasSuffix(fpath, ext)
	}
	return suffix(".ent.gz") || suffix(".pdb") || suffix(".ent")
}
