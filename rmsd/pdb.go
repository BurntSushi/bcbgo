package rmsd

import (
	"fmt"

	"github.com/BurntSushi/bcbgo/pdb"
)

// PDB is a convenience function for computing the RMSD between two sets of
// residues, where each set is take from a chain of a PDB entry. Note that RMSD
// is only computed using carbon-alpha atoms.
//
// Each set of atoms to be used is specified by a four-tuple: a PDB entry file,
// a chain identifier, and the start and end residue numbers to use as a range.
// (Where the range is inclusive.)
//
// An error will be returned if: chainId{1,2} does not correspond to a chain
// in entry{1,2}. The ranges specified by start{1,2}-end{1,2} are not valid.
// The ranges specified by start{1,2}-end{1,2} do not correspond to precisely
// the same number of carbon-alpha atoms.
func PDB(entry1 *pdb.Entry, chainId1 byte, start1, end1 int,
	entry2 *pdb.Entry, chainId2 byte, start2, end2 int) (float64, error) {

	chain1, ok := entry1.Chains[chainId1]
	if !ok {
		return 0.0, fmt.Errorf("The chain '%c' could not be found in '%s'.",
			chainId1, entry1.Name())
	}
	chain2, ok := entry2.Chains[chainId2]
	if !ok {
		return 0.0, fmt.Errorf("The chain '%c' could not be found in '%s'.",
			chainId2, entry2.Name())
	}

	// In order to fetch the appropriate carbon-alpha atoms, we need to
	// traverse each chain's carbon-alpha atom slice and pick only the carbon
	// alpha atoms with residue indices in the range specified.
	struct1 := make(pdb.Atoms, 0, max(0, end1-start1+1))
	struct2 := make(pdb.Atoms, 0, max(0, end2-start2+1))
	for _, atom := range chain1.CaAtoms {
		if atom.ResidueInd >= start1 && atom.ResidueInd <= end1 {
			struct1 = append(struct1, atom)
		}
	}
	for _, atom := range chain2.CaAtoms {
		if atom.ResidueInd >= start2 && atom.ResidueInd <= end2 {
			struct2 = append(struct2, atom)
		}
	}

	// Verify that neither of the atom sets is 0.
	if len(struct1) == 0 {
		return 0.0, fmt.Errorf("The range '%d-%d' (for chain %c in %s) does "+
			"not correspond to any carbon-alpha ATOM records.",
			start1, end1, chainId1, entry1.Name())
	}
	if len(struct2) == 0 {
		return 0.0, fmt.Errorf("The range '%d-%d' (for chain %c in %s) does "+
			"not correspond to any carbon-alpha ATOM records.",
			start2, end2, chainId2, entry2.Name())
	}

	// If we don't have the same number of atoms from each chain, we can't
	// compute RMSD.
	if len(struct1) != len(struct2) {
		return 0.0, fmt.Errorf("The range '%d-%d' (%d ATOM records for chain "+
			"%c in %s) does not correspond to the same number of carbon-alpha "+
			"atoms as the range '%d-%d' (%d ATOM records for chain %c in %s). "+
			"It is possible that the PDB file does not contain a carbon-alpha "+
			"atom for every residue index in the ranges.",
			start1, end1, len(struct1), chainId1, entry1.Name(),
			start2, end2, len(struct2), chainId2, entry2.Name())
	}

	// We're good to go...
	return RMSD(struct1, struct2), nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
