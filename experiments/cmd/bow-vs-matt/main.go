// test bow-vs-matt compares the structural neighbors detected by a search
// on BOW vectors verus the structural neighbors detected by the Matt
// structural aligner.
//
// The test only produces the output of the comparison, but does not say how
// similar they are.
//
// The comparison is that of ordering. Namely, given some protein chain A,
// find the closest structural neighbors returned by two different methods:
// structural alignment and BOW searching.
//
// This is done by first creating a BOW database from the given list of PDB
// files. (This requires that 'create-pdb-db' is in your PATH.) Then, output
// a file for every protein chain that contains a ordering of all structural
// neighbors using both search techniques.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/BurntSushi/bcbgo/apps/matt"
	"github.com/BurntSushi/bcbgo/bow"
	"github.com/BurntSushi/bcbgo/cmd/util"
)

type results []result

type result struct {
	entry   string
	chain   byte
	results bow.SearchResult
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	util.FlagUse("cpu")
	util.FlagParse("database-path frag-lib-dir query-pdb-file "+
		"[query-pdb-file ...]", "")
	util.AssertLeastNArg(2)
}

func main() {
	dbPath := util.Arg(0)
	fragLibDir := util.Arg(1)
	pdbFiles := flag.Args()[2:]

	util.Assert(createBowDb(dbPath, fragLibDir, pdbFiles))

	db, err := bow.OpenDB(dbPath)
	util.Assert(err)

	bowOpts := bow.SearchDefault
	bowOpts.Limit = 200
	mattOpts := matt.DefaultConfig
	mattOpts.Verbose = false

	chains := createChains(pdbFiles)
	mattArgs := createMattArgs(chains)

	tabw := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
	header := []byte(
		"BOW entry\t" +
			"BOW chain\t" +
			"BOW dist\t" +
			"Matt entry\t" +
			"Matt chain\t" +
			"Matt dist\n")
	for i, chain := range chains {
		marg := mattArgs[i]

		bowOrdered := getBowOrdering(db, bowOpts, chain)
		mattOrdered := getMattOrdering(mattOpts, marg, mattArgs)

		fmt.Printf("Ordering for %s (chain %c)\n",
			chain.Entry.IdCode, chain.Ident)

		compared := comparison([2]ordering{bowOrdered, mattOrdered})
		tabw.Write(header)
		tabw.Write([]byte(compared.String()))
		tabw.Flush()
		fmt.Println("\n")
	}

	util.Assert(db.Close())
}

func createBowDb(dbPath string, fragLibDir string, pdbFiles []string) error {
	if _, err := os.Stat(dbPath); err == nil || os.IsExist(err) {
		return nil
	}

	args := []string{dbPath, fragLibDir}
	args = append(args, pdbFiles...)
	cmd := exec.Command("bowmk", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error running '%s': %s.",
			strings.Join(cmd.Args, " "), err)
	}
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
