package hhfrag

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

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

// An HHfrag FragmentMap maps each N-window (with starting positions separated
// by K-residues) of a query sequence to a set of HMM/PDB fragments.
type FragmentMap struct {
	WindowSizeMin   int
	WindowSizeMax   int
	WindowIncrement int
	Map             map[int]Fragments
	Blits           bool
}

func NewFragmentMap(blits bool, increment, min, max int) FragmentMap {
	return FragmentMap{
		WindowSizeMin:   min,
		WindowSizeMax:   max,
		WindowIncrement: increment,
		Map:             make(map[int]Fragments, 50),
		Blits:           blits,
	}
}

func (m FragmentMap) Fill(
	pdbDb PDBDatabase, seqDb hhsuite.Database, query string) error {

	return nil
}

type Fragments []Fragment

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
	pdbEntry, err := pdb.New(path.Join(pdbDb.PDB(), pdbName))
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

	// Things get tricky here. The alpha-carbon ATOM records don't necessarily
	// correspond to SEQRES residues. So we need to use the AtomResidueStart
	// and AtomResidueEnd to make sure we're looking at the right Ca atoms.
	if ts < chain.AtomResidueStart || te > chain.AtomResidueEnd {
		return Fragment{},
			fmt.Errorf("The template sequence (%d, %d) is not in the ATOM "+
				"residue range (%d, %d)",
				ts, te, chain.AtomResidueStart, chain.AtomResidueEnd)
	}
	atoms := make(pdb.Atoms, te-ts+1)

	// One again, we copy to avoid pinning memory.
	as, ae := ts-chain.AtomResidueStart, te-chain.AtomResidueStart+1
	copy(atoms, chain.CaAtoms[as:ae])

	return Fragment{
		Query:    qs.Slice(hit.QueryStart-1, hit.QueryEnd),
		Template: tseq,
		Hit:      hit,
		CaAtoms:  atoms,
	}, nil
}

func getTemplatePdbName(hitName string) string {
	return strings.SplitN(strings.TrimSpace(hitName), " ", 2)[0]
}
