package hhfrag

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/io/fasta"
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

// An HHfrag FragmentMap maps each N-window (with starting positions separated
// by K-residues) of a query sequence to a set of HMM/PDB fragments.
type FragmentMap struct {
	WindowMin       int
	WindowMax       int
	WindowIncrement int
	Map             map[int]Fragments
	Blits           bool
}

func NewFragmentMap(blits bool, increment, min, max int) FragmentMap {
	return FragmentMap{
		WindowMin:       min,
		WindowMax:       max,
		WindowIncrement: increment,
		Map:             make(map[int]Fragments, 50),
		Blits:           blits,
	}
}

func (m FragmentMap) Fill(
	pdbDb PDBDatabase, seqDb hhsuite.Database, query string) error {

	fquery, err := os.Open(query)
	if err != nil {
		return err
	}

	seqs, err := fasta.NewReader(fquery).ReadAll()
	if err != nil {
		return err
	} else if len(seqs) == 0 {
		return fmt.Errorf("No sequences found in '%s'.", query)
	} else if len(seqs) > 1 {
		return fmt.Errorf("%d sequences found in '%s'. Expected only 1.",
			len(seqs), query)
	}
	qseq := seqs[0]

	queryHHM, err := hhsuite.BuildHHM(
		hhsuite.HHBlitsDefault, hhsuite.HHMakePseudo, seqDb, query)

	for i := 0; i <= qseq.Len()-m.WindowMin; i += 3 {
		var best Fragments
		for j := m.WindowMin; j < m.WindowMax && (i+j) <= qseq.Len(); j++ {
			part := queryHHM.Slice(i, i+j)
			frags, err := FindFragments(pdbDb, part, qseq, m.Blits)
			if err != nil {
				return err
			}

			if best == nil || frags.better(best) {
				best = frags
			}
		}
		m.Map[i] = best
	}
	return nil
}

type Fragments []Fragment

// better returns true if f1 is 'better' than f2. Otherwise false.
func (f1 Fragments) better(f2 Fragments) bool {
	return len(f1) > len(f2)
}

func FindFragments(pdbDb PDBDatabase,
	part *hhm.HHM, qs seq.Sequence, blits bool) (Fragments, error) {

	hhmFile, err := ioutil.TempFile("", "bcbgo-hhfrag-hhm")
	if err != nil {
		return nil, err
	}
	defer os.Remove(hhmFile.Name())
	hhmName := hhmFile.Name()

	if err := hhm.Write(hhmFile, part); err != nil {
		return nil, err
	}

	var results *hhr.HHR
	if blits {
		results, err = hhsuite.HHBlitsDefault.Run(pdbDb.HHsuite(), hhmName)
	} else {
		results, err = hhsuite.HHSearchDefault.Run(pdbDb.HHsuite(), hhmName)
	}
	if err != nil {
		return nil, err
	}

	frags := make(Fragments, len(results.Hits))
	for i, hit := range results.Hits {
		frag, err := NewFragment(pdbDb, qs, hit)
		if err != nil {
			return nil, err
		}
		frags[i] = frag
	}
	return frags, nil
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

	// Things get tricky here. The alpha-carbon ATOM records don't necessarily
	// correspond to SEQRES residues. So we need to use the AtomResidueStart
	// and AtomResidueEnd to make sure we're looking at the right Ca atoms.
	// Note that we might still want to look at this hit, so we simply set
	// the CaAtoms field to nil, which gives it a "corrupt" label.
	if ts < chain.AtomResidueStart || te > chain.AtomResidueEnd {
		return frag, nil
	}

	// One again, we copy to avoid pinning memory.
	frag.CaAtoms = make(pdb.Atoms, te-ts+1)
	as, ae := ts-chain.AtomResidueStart, te-chain.AtomResidueStart+1
	copy(frag.CaAtoms, chain.CaAtoms[as:ae])

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
