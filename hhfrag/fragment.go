package hhfrag

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"text/tabwriter"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/io/hhm"
	"github.com/BurntSushi/bcbgo/io/hhr"
	"github.com/BurntSushi/bcbgo/io/pdb"
	"github.com/BurntSushi/bcbgo/seq"
)

type PDBDatabase hhsuite.Database

func (db PDBDatabase) HHsuite() hhsuite.Database {
	resolved := hhsuite.Database(db).Resolve()
	dbName := path.Base(resolved)
	return hhsuite.Database(path.Join(resolved, dbName))
}

func (db PDBDatabase) PDB() string {
	resolved := hhsuite.Database(db).Resolve()
	return path.Join(resolved, "pdb")
}

type Fragments struct {
	Frags      []Fragment
	Start, End int
}

// better returns true if f1 is 'better' than f2. Otherwise false.
func (f1 Fragments) better(f2 Fragments) bool {
	return len(f1.Frags) >= len(f2.Frags)
}

func (frags Fragments) Write(w io.Writer) {
	tabw := tabwriter.NewWriter(w, 0, 4, 4, ' ', 0)
	fmt.Fprintln(tabw, "Hit\tQuery\tTemplate\tProb\tCorrupt")
	for _, frag := range frags.Frags {
		var corruptStr string
		if frag.IsCorrupt() {
			corruptStr = "\tcorrupt"
		}
		fmt.Fprintf(tabw, "%s\t(%d-%d)\t(%d-%d)\t%f%s\n",
			frag.Template.Name,
			frag.Hit.QueryStart, frag.Hit.QueryEnd,
			frag.Hit.TemplateStart, frag.Hit.TemplateEnd,
			frag.Hit.Prob,
			corruptStr)
	}
	tabw.Flush()
}

func FindFragments(pdbDb PDBDatabase, blits bool,
	queryHHM *hhm.HHM, qs seq.Sequence, start, end int) (*Fragments, error) {

	pre := fmt.Sprintf("bcbgo-hhfrag-hhm-%d-%d_", start, end)
	hhmFile, err := ioutil.TempFile("", pre)
	if err != nil {
		return nil, err
	}
	defer os.Remove(hhmFile.Name())
	hhmName := hhmFile.Name()

	if err := hhm.Write(hhmFile, queryHHM.Slice(start, end)); err != nil {
		return nil, err
	}

	var results *hhr.HHR
	if blits {
		conf := hhsuite.HHBlitsDefault
		conf.CPUs = 1
		results, err = conf.Run(pdbDb.HHsuite(), hhmName)
	} else {
		conf := hhsuite.HHSearchDefault
		conf.CPUs = 1
		results, err = conf.Run(pdbDb.HHsuite(), hhmName)
	}
	if err != nil {
		return nil, err
	}

	frags := make([]Fragment, len(results.Hits))
	for i, hit := range results.Hits {
		hit.QueryStart += start
		hit.QueryEnd += start
		frag, err := NewFragment(pdbDb, qs, hit)
		if err != nil {
			return nil, err
		}
		frags[i] = frag
	}
	return &Fragments{
		Frags: frags,
		Start: start,
		End:   end,
	}, nil
}

// An HHfrag Fragment corresponds to a match between a portion of a query
// HMM and a portion of a template HMM. The former is represented as a slice
// of a regular sequence, where the latter is represented as an hhsuite hit
// and a list of alpha-carbon atoms corresponding to the matched region.
type Fragment struct {
	Query    seq.Sequence
	Template seq.Sequence
	Hit      hhr.Hit
	CaAtoms  pdb.Atoms
}

// IsCorrupt returns true when a particular fragment could not be paired
// with alpha-carbon positions for every residue in the template strand.
// (This problem stems from the fact that we use SEQRES records for sequence
// information, but not all residues in SEQRES have alpha-carbon ATOM records
// associated with them.)
func (frag Fragment) IsCorrupt() bool {
	return frag.CaAtoms == nil
}

// NewFragment constructs a new fragment from a full query sequence and the
// hit from the HHR file.
//
// Since NewFragment requires access to the raw PDB alpha-carbon atoms (and
// the sequence) of the template hit, you'll also need to pass a path to the
// PDB database. (Which is a directory containing a flat list of all
// PDB files used to construct the corresponding hhblits database.) This
// database is usually located inside the 'pdb' directory contained in the
// corresponding hhsuite database. i.e., $HHLIB/data/pdb-select25/pdb
func NewFragment(
	pdbDb PDBDatabase, qs seq.Sequence, hit hhr.Hit) (Fragment, error) {

	pdbName := getTemplatePdbName(hit.Name)
	pdbEntry, err := pdb.New(path.Join(
		pdbDb.PDB(), fmt.Sprintf("%s.pdb", pdbName)))
	if err != nil {
		return Fragment{}, err
	}

	// Load in the sequence from the PDB file using the SEQRES residues.
	ts, te := hit.TemplateStart, hit.TemplateEnd
	chain := pdbEntry.OneChain()
	tseq := seq.Sequence{
		Name:     pdbName,
		Residues: make([]seq.Residue, te-ts+1),
	}

	// We copy here to avoid pinning pdb.Entry objects.
	copy(tseq.Residues, chain.Sequence[ts-1:te])

	frag := Fragment{
		Query:    qs.Slice(hit.QueryStart-1, hit.QueryEnd),
		Template: tseq,
		Hit:      hit,
		CaAtoms:  nil,
	}

	// We designate "corrupt" if the query/template hit regions are of
	// different length. i.e., we don't allow gaps (yet).
	// BUG(burntsushi): Fragments with gaps are marked as corrupt.
	if hit.QueryEnd-hit.QueryStart != hit.TemplateEnd-hit.TemplateStart {
		return frag, nil
	}

	// We also designate "corrupt" if there are any gaps in our alpha-carbon
	// atom list.
	atoms := chain.CaAtomSlice(ts-1, te)
	if atoms == nil {
		return frag, nil
	}

	// One again, we copy to avoid pinning memory.
	frag.CaAtoms = make(pdb.Atoms, len(atoms))
	copy(frag.CaAtoms, atoms)

	return frag, nil
}

func getTemplatePdbName(hitName string) string {
	return strings.SplitN(strings.TrimSpace(hitName), " ", 2)[0]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
