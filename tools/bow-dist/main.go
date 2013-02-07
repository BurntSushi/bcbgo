package main

import (
	"fmt"
	"math"

	"github.com/BurntSushi/bcbgo/cmd/util"
)

func init() {
	util.FlagParse("bow1 bow2", "")
	util.AssertNArg(2)
}

func main() {
	bow1 := util.BOWRead(util.Arg(0))
	bow2 := util.BOWRead(util.Arg(1))
	fmt.Printf("%0.4f\n", math.Abs(bow1.Cosine(bow2)))
}
