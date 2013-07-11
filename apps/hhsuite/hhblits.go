package hhsuite

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/BurntSushi/cmd"

	"github.com/TuftsBCB/io/hhm"
	"github.com/TuftsBCB/io/hhr"
)

type HHBlitsConfig struct {
	Exec       string
	CPUs       int
	Iterations int
	MAct       float64
	OutA3M     string

	// When true, the 'hhblits' stdout and stderr will be mapped to the
	// current processes' stdout and stderr.
	Verbose bool
}

var HHBlitsDefault = HHBlitsConfig{
	Exec:       "hhblits",
	CPUs:       runtime.NumCPU(),
	Iterations: 2,
	MAct:       0.35,
	OutA3M:     "",
	Verbose:    false,
}

// Run will execute HHblits using the given configuration, database and query
// file path. The query can be a path to a fasta file, A3M file or HHM file.
// (As per the '-i' flag for hhblits.)
func (conf HHBlitsConfig) Run(db Database, query string) (*hhr.HHR, error) {
	// If the database is old style, it cannot be used with hhblits.
	if db.isOldStyle() {
		return nil, fmt.Errorf("An old-style database '%s' cannot be used "+
			"with hhblits. It can only be used with hhsearch.", db)
	}

	hhrFile, err := ioutil.TempFile("", "bcbgo-hhr")
	if err != nil {
		return nil, err
	}
	defer os.Remove(hhrFile.Name())
	defer hhrFile.Close()

	args := []string{
		"-cpu", fmt.Sprintf("%d", conf.CPUs),
		"-i", query,
		"-d", db.Resolve(),
		"-n", fmt.Sprintf("%d", conf.Iterations),
		"-mact", fmt.Sprintf("%f", conf.MAct),
		"-o", hhrFile.Name(),
	}
	if len(conf.OutA3M) > 0 {
		args = append(args, []string{"-oa3m", conf.OutA3M}...)
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

// BuildHHM is a convenience function for building an HHM file (with pseudo
// count correction for emission/gaps) from a single sequence FASTA file.
// Namely, hhblits and hhmake are the configurations for each program.
// db is the database to use to generate an MSA (usually 'nr20' or 'uniprot20'),
// and query is a file path pointing to a single sequence FASTA file.
//
// N.B. The hhblits configuration is modified to include a temporary A3M file,
// which is the output of hhblits and the input to hhmake. Thus, BuildHHM will
// panic if the hhblits configuration contains a non-empty A3M output file name.
func BuildHHM(hhblits HHBlitsConfig, hhmake HHMakeConfig,
	db Database, query string) (*hhm.HHM, error) {

	if len(hhblits.OutA3M) > 0 {
		panic("hhblits configuration for BuildHHM should have empty OutA3M.")
	}

	a3mFile, err := ioutil.TempFile("", "bcbgo-a3m")
	if err != nil {
		return nil, err
	}
	defer os.Remove(a3mFile.Name())
	defer a3mFile.Close()

	hhblits.OutA3M = a3mFile.Name()
	_, err = hhblits.Run(db, query)
	if err != nil {
		return nil, err
	}
	return hhmake.Run(a3mFile.Name())
}
