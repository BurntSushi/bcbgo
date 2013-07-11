package hhsuite

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/BurntSushi/cmd"

	"github.com/TuftsBCB/io/hhm"
)

type HHMakeConfig struct {
	Exec string
	PCM  int
	PCA  float64
	PCB  float64
	PCC  float64
	GapB float64
	GapD float64
	GapE float64
	GapF float64
	GapG float64
	GapI float64

	// When true, the 'hhmake' stdout and stderr will be mapped to the
	// current processes' stdout and stderr.
	Verbose bool
}

var HHMakePseudo = HHMakeConfig{
	Exec: "hhmake",
	PCM:  4,
	PCA:  2.5,
	PCB:  0.5,
	PCC:  1.0,
	GapB: 1.0,
	GapD: 0.15,
	GapE: 1.0,
	GapF: 0.6,
	GapG: 0.6,
	GapI: 0.6,
}

// Run will execute HHmake using the given configuration and query file path.
// The query should be a file path pointing to an MSA file (fasta, a2m or
// a3m) or an hhm file. It should NOT be just a single sequence.
//
// If you need to build an HHM from a single sequence, use the convenience
// function BuildHHM.
func (conf HHMakeConfig) Run(query string) (*hhm.HHM, error) {
	hhmFile, err := ioutil.TempFile("", "bad-bcbgo-hhm")
	if err != nil {
		return nil, err
	}
	defer os.Remove(hhmFile.Name())
	defer hhmFile.Close()

	emission := strings.Fields(fmt.Sprintf(
		"-pcm %d -pca %f -pcb %f -pcc %f",
		conf.PCM, conf.PCA, conf.PCB, conf.PCC))
	transition := strings.Fields(fmt.Sprintf(
		"-gapb %f -gapd %f -gape %f -gapf %f -gapg %f -gapi %f",
		conf.GapB, conf.GapD, conf.GapE, conf.GapF, conf.GapG, conf.GapI))

	args := []string{
		"-i", query,
		"-o", hhmFile.Name(),
	}
	args = append(args, emission...)
	args = append(args, transition...)

	c := cmd.New(conf.Exec, args...)
	if conf.Verbose {
		fmt.Fprintf(os.Stderr, "\n%s\n", c)
		c.Cmd.Stdout = os.Stdout
		c.Cmd.Stderr = os.Stderr
	}
	if err := c.Run(); err != nil {
		return nil, err
	}
	return hhm.Read(hhmFile)
}
