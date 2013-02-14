package main

import (
	"flag"
	"fmt"
	"path"
	"strings"

	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/io/hhr"
	"github.com/BurntSushi/bcbgo/io/pdb"
	"github.com/BurntSushi/bcbgo/rmsd"
	"github.com/BurntSushi/bcbgo/seq"
)

var (
	flagRmsd = 1.5
)

func init() {
	flag.Float64Var(&flagRmsd, "rmsd", flagRmsd,
		"The RMSD cut-off to use to determine true positives.")

	util.FlagUse("pdb-dir")
	util.FlagParse("fmap-file", "")
	util.AssertNArg(1)
}

func main() {
	fmapPath := util.Arg(0)

	fmap := util.FmapRead(fmapPath)
	qchain := getPdbChain(fmapPath)
	stats := newSequenceStats(qchain.Sequence)

	total, trueps := 0, 0
	qcorrupt, tcorrupt := 0, 0
	for _, frags := range fmap.Segments {
		for _, frag := range frags.Frags {
			hit := frag.Hit

			if frag.IsCorrupt() {
				tcorrupt += 1
				stats.incTCorrupt(hit)
				continue
			}

			qatoms := qchain.SequenceCaAtomSlice(hit.QueryStart-1, hit.QueryEnd)
			if qatoms == nil {
				qcorrupt += 1
				stats.incQCorrupt(hit)
				continue
			}

			if len(qatoms) != len(frag.CaAtoms) {
				util.Fatalf("Uncomparable lengths. Query is (%d, %d) while "+
					"template is (%d, %d). Length of query CaAtoms: %d, "+
					"length of template CaAtoms: %d",
					hit.QueryStart, hit.QueryEnd,
					hit.TemplateStart, hit.TemplateEnd,
					len(qatoms), len(frag.CaAtoms))
			}

			if rmsd.QCRMSD(qatoms, frag.CaAtoms) <= flagRmsd {
				trueps += 1
				stats.incTruePs(hit)
			}
			total += 1
			stats.incTotal(hit)
		}
	}

	coveredResidues := 0
	for _, resStats := range stats {
		if resStats.trueps >= 1 {
			coveredResidues += 1
		}
	}
	coverage := float64(coveredResidues) / float64(len(qchain.Sequence))

	fmt.Printf("RMSDThreshold: %f\n", flagRmsd)
	fmt.Printf("TotalFragments: %d\n", total)
	fmt.Printf("TruePositives: %d\n", trueps)
	fmt.Printf("Precision: %f\n", float64(trueps)/float64(total))
	fmt.Printf("CorruptQuery: %d\n", qcorrupt)
	fmt.Printf("CorruptTemplate: %d\n", tcorrupt)
	fmt.Printf("TotalResidues: %d\n", len(qchain.Sequence))
	fmt.Printf("CoveredResidues: %d\n", coveredResidues)
	fmt.Printf("Coverage: %f\n", coverage)
}

type sequenceStats []residueStats

// Per residue stats.
type residueStats struct {
	residue  seq.Residue
	total    int
	trueps   int
	qcorrupt int
	tcorrupt int
}

func newSequenceStats(residues []seq.Residue) sequenceStats {
	m := make(sequenceStats, len(residues))
	for i, residue := range residues {
		m[i] = residueStats{
			residue:  residue,
			total:    0,
			trueps:   0,
			qcorrupt: 0,
			tcorrupt: 0,
		}
	}
	return m
}

func (ss sequenceStats) incTotal(hit hhr.Hit) {
	for i := hit.QueryStart - 1; i < hit.QueryEnd; i++ {
		ss[i].total += 1
	}
}

func (ss sequenceStats) incTruePs(hit hhr.Hit) {
	for i := hit.QueryStart - 1; i < hit.QueryEnd; i++ {
		ss[i].trueps += 1
	}
}

func (ss sequenceStats) incTCorrupt(hit hhr.Hit) {
	for i := hit.QueryStart - 1; i < hit.QueryEnd; i++ {
		ss[i].tcorrupt += 1
	}
}

func (ss sequenceStats) incQCorrupt(hit hhr.Hit) {
	for i := hit.QueryStart - 1; i < hit.QueryEnd; i++ {
		ss[i].qcorrupt += 1
	}
}

func getPdbChain(fp string) *pdb.Chain {
	b := path.Base(fp)
	if !strings.HasSuffix(b, ".fmap") {
		util.Fatalf("Expected file named 'something.fmap' but got '%s'.", b)
	}
	idAndChain := b[0 : len(b)-5]
	if len(idAndChain) != 5 {
		util.Fatalf("Expected 4-letter PDB id concatenated with 1-letter "+
			"chain identifier, but got '%s' instead.", idAndChain)
	}

	pdbName := idAndChain[0:4]
	chainId := idAndChain[4]
	pdbCat := idAndChain[1:3]
	pdbFile := fmt.Sprintf("pdb%s.ent.gz", pdbName)

	pdbPath := path.Join(util.FlagPdbDir, pdbCat, pdbFile)

	entry := util.PDBRead(pdbPath)
	chain := entry.Chain(chainId)
	if chain == nil {
		util.Fatalf("Could not find chain '%c' in PDB entry '%s'.",
			chainId, pdbPath)
	}
	return chain
}
