package fragbag

import (
	"encoding/gob"
	"fmt"
	"io"

	"github.com/BurntSushi/bcbgo/seq"
)

// SequenceLibrary represents a Fragbag sequence fragment library.
// Fragbag fragment libraries are fixed both in the number of fragments and in
// the size of each fragment.
type SequenceLibrary struct {
	Ident        string
	Fragments    []SequenceFragment
	FragmentSize int
}

// NewSequenceLibrary initializes a new Fragbag sequence library with the
// given name. It is not written to disk until Save is called.
func NewSequenceLibrary(name string) *SequenceLibrary {
	lib := new(SequenceLibrary)
	lib.Ident = name
	return lib
}

// Add adds a sequence fragment to the library, where a sequence fragment
// corresponds to a profile of log-odds scores for each amino acid.
// The first call to Add may contain any number of columns in the profile.
// All subsequent adds must contain the same number of columns as the first.
func (lib *SequenceLibrary) Add(prof *seq.Profile) error {
	if lib.Fragments == nil || len(lib.Fragments) == 0 {
		frag := SequenceFragment{0, prof}
		lib.Fragments = append(lib.Fragments, frag)
		lib.FragmentSize = prof.Len()
		return nil
	}

	frag := SequenceFragment{len(lib.Fragments), prof}
	if lib.FragmentSize != prof.Len() {
		return fmt.Errorf("Fragment %d has length %d; expected length %d.",
			frag.FragNumber(), prof.Len(), lib.FragmentSize)
	}
	lib.Fragments = append(lib.Fragments, frag)
	return nil
}

// Save saves the full fragment library to the writer provied.
func (lib *SequenceLibrary) Save(w io.Writer) error {
	enc := gob.NewEncoder(w)
	return enc.Encode(*lib)
}

// Open loads an existing structure fragment library from the reader provided.
func OpenSequenceLibrary(r io.Reader) (*SequenceLibrary, error) {
	var lib *SequenceLibrary

	dec := gob.NewDecoder(r)
	if err := dec.Decode(&lib); err != nil {
		return nil, err
	}
	return lib, nil
}

// Size returns the number of fragments in the library.
func (lib *SequenceLibrary) Size() int {
	return len(lib.Fragments)
}

// String returns a string with the name of the library, the number of
// fragments in the library and the size of each fragment.
func (lib *SequenceLibrary) String() string {
	return fmt.Sprintf("%s (%d, %d)",
		lib.Ident, len(lib.Fragments), lib.FragmentSize)
}

func (lib *SequenceLibrary) Name() string {
	return lib.Ident
}

// Best returns the number of the fragment that best corresponds
// to the string of amino acids provided.
// The length of `sequence` must be equivalent to the fragment size.
func (lib *SequenceLibrary) Best(s seq.Sequence) int {
	panic("unimplemented")
}

// Fragment corresponds to a single sequence fragment in a fragment library.
// It holds the fragment number identifier and embeds a sequence profile.
type SequenceFragment struct {
	Number int
	*seq.Profile
}

func (frag *SequenceFragment) FragNumber() int {
	return frag.Number
}

func (frag *SequenceFragment) String() string {
	return fmt.Sprintf("> %d", frag.Number)
}
