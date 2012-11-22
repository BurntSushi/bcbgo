package hhsuite

import (
	"io/ioutil"
	"os"
	"runtime"

	"github.com/BurntSushi/cmd"

	"github.com/BurntSushi/bcbgo/io/hhr"
)

type HHBlitsConfig struct {
	Exec       string
	CPUs       int
	Iterations int
	MAct       float64
	OutA3M     string
}

var HHBlitsDefault = HHBlitsConfig{
	Exec:       "hhblits",
	CPUs:       runtime.NumCPU(),
	Iterations: 2,
	MAct:       0.35,
	OutA3M:     "",
}

// Run will execute HHblits using the given configuration, database and query
// file path. The query can be a path to a fasta file, A3M file or HHM file.
// (As per the '-i' flag for hhblits.)
func (conf HHBlitsConfig) Run(db Database, query string) (*hhr.HHR, error) {
	hhrFile, err := ioutil.TempFile("", "bcbgo-hhr")
	if err != nil {
		return nil, err
	}
	defer os.Remove(hhrFile.Name())

	args := []string{
		"-cpu", fmt.Sprintf("%d", conf.CPUs),
		"-i", query,
		"-d", db.Resolve(),
		"-n", conf.Iterations,
		"-mact", fmt.Sprintf("%f", conf.MAct),
		"-o", hhrFile.Name(),
	}
	if len(conf.OutA3M) > 0 {
		args = append(args, []string{"-oa3m", conf.OutA3M}...)
	}
	err = cmd.New(conf.Path, args...).Run()
	if err != nil {
		return nil, err
	}

	results, err := hhr.Read(hhrFile)
	if err != nil {
		return nil, err
	}
	return results, nil
}
