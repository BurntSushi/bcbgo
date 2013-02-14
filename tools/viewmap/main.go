package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/bcbgo/cmd/util"
)

func init() {
	util.FlagParse("fmap-file", "")
	util.AssertNArg(1)
}

func main() {
	fmap := util.FmapRead(util.Arg(0))
	fmt.Printf("%s\n\n", fmap.Name)
	for _, frags := range fmap.Segments {
		fmt.Printf("\nSEGMENT: %d %d (%d)\n",
			frags.Start, frags.End, len(frags.Frags))
		frags.Write(os.Stdout)
	}
}
