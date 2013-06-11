package main

import (
	"fmt"

	"github.com/BurntSushi/bcbgo/cmd/util"
)

func init() {
	util.FlagParse("seq-lib", "")
	util.AssertNArg(1)
}

func main() {
	libPath := util.Arg(0)

	seqLib := util.SequenceLibrary(libPath)
	fmt.Println(seqLib)
	for _, frag := range seqLib.Fragments {
		fmt.Printf("%s\n\n", frag.String())
	}
}
