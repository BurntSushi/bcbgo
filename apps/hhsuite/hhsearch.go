package hhsuite

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/BurntSushi/cmd"

	"github.com/BurntSushi/bcbgo/io/hhr"
)

type HHSearchConfig struct {
	Exec string
	CPUs int

	// When true, the 'hhsearch' stdout and stderr will be mapped to the
	// current processes' stdout and stderr.
	Verbose bool
}

var HHSearchDefault = HHSearchConfig{
	Exec:    "hhsearch",
	CPUs:    runtime.NumCPU(),
	Verbose: false,
}

// Run will execute HHsearch using the given configuration, database and query
// file path. The query can be a path to a fasta file, A3M file or HHM file.
// (As per the '-i' flag for hhsearch.)
func (conf HHSearchConfig) Run(db Database, query string) (*hhr.HHR, error) {
	hhrFile, err := ioutil.TempFile("", "bcbgo-hhr")
	if err != nil {
		return nil, err
	}
	defer os.Remove(hhrFile.Name())

	args := []string{
		"-cpu", fmt.Sprintf("%d", conf.CPUs),
		"-i", query,
		"-d", db.Resolve(),
		"-o", hhrFile.Name(),
	}

	c := cmd.New(conf.Exec, args...)
	if conf.Verbose {
		fmt.Fprintf(os.Stderr, "\n%s\n", c)
		c.Cmd.Stdout = os.Stdout
		c.Cmd.Stderr = os.Stderr
	}
	if err := c.Run(); err != nil {
		return nil, err
	}
	return hhr.Read(hhrFile)
}
