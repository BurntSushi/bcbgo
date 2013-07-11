package main

import (
	"fmt"

	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/TuftsBCB/io/pdb"
)

func init() {
	u := "pdb-file chain-id start stop pdb-file chain-id start stop"
	util.FlagParse(u, "")
	util.AssertNArg(8)
}

func main() {
	pdbf1, chain1, s1, e1 := util.Arg(0), util.Arg(1), util.Arg(2), util.Arg(3)
	pdbf2, chain2, s2, e2 := util.Arg(4), util.Arg(5), util.Arg(6), util.Arg(7)

	entry1 := util.PDBRead(pdbf1)
	entry2 := util.PDBRead(pdbf2)

	s1n, e1n := util.ParseInt(s1), util.ParseInt(e1)
	s2n, e2n := util.ParseInt(s2), util.ParseInt(e2)

	r, err := pdb.RMSD(
		entry1, chain1[0], s1n, e1n, entry2, chain2[0], s2n, e2n)
	util.Assert(err)
	fmt.Println(r)
}
