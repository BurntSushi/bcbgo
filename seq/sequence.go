package seq

import (
	"fmt"
)

// A Sequence corresponds to any kind of biological sequence: DNA, RNA, amino
// acid, secondary structure, etc.
type Sequence struct {
	Name     string
	Residues []Residue
}

// A Residue corresponds to a single entry in a sequence.
type Residue byte

// Copy returns a deep copy of the sequence.
func (s Sequence) Copy() Sequence {
	residues := make([]Residue, len(s.Residues))
	copy(residues, s.Residues)
	return Sequence{
		Name:     fmt.Sprintf("%s", s.Name),
		Residues: residues,
	}
}

// Len returns the number of residues in the sequence.
func (s Sequence) Len() int {
	return len(s.Residues)
}

// HMMState returns the HMMState of a particular residue in a sequence.
// Residues in [A-Z] are match states. Residues matching '-' are deletion
// states. Residues equal to '.' or in [a-z] are insertion states.
//
// A residue corresponding to any other value will panic.
//
// The pre-condition here is that 'r' is a residue from a sequence from an
// A2M format. (N.B. MSA's formed from A3M and FASTA formatted files are
// repsented as A2M format, so MSA's read from A3M/FASTA files are OK.)
func (r Residue) HMMState() HMMState {
	switch {
	case r == '-':
		return Deletion
	case r == '.':
		return Insertion
	case r >= 'a' && r <= 'z':
		return Insertion
	case r >= 'A' && r <= 'Z':
		return Match
	}
	panic(fmt.Sprintf("Unknown residue: %c", r))
}
