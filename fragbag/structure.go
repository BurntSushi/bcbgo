package fragbag

import (
	"encoding/gob"
	"fmt"
	"io"
	"strings"

	"github.com/TuftsBCB/structure"
)

// StructureLibrary represents a Fragbag structural fragment library.
// Fragbag fragment libraries are fixed both in the number of fragments and in
// the size of each fragment.
type StructureLibrary struct {
	Ident        string
	Fragments    []StructureFragment
	FragmentSize int
}

// NewStructureLibrary initializes a new Fragbag structure library with the
// given name. It is not written to disk until Save is called.
func NewStructureLibrary(name string) *StructureLibrary {
	lib := new(StructureLibrary)
	lib.Ident = name
	return lib
}

// Add adds a structural fragment to the library. The first call to Add may
// contain any number of coordinates. All subsequent adds must contain the
// same number of coordinates as the first.
func (lib *StructureLibrary) Add(coords []structure.Coords) error {
	if lib.Fragments == nil || len(lib.Fragments) == 0 {
		frag := StructureFragment{0, coords}
		lib.Fragments = append(lib.Fragments, frag)
		lib.FragmentSize = len(coords)
		return nil
	}

	frag := StructureFragment{len(lib.Fragments), coords}
	if lib.FragmentSize != len(coords) {
		return fmt.Errorf("Fragment %d has length %d; expected length %d.",
			frag.FragNumber(), len(coords), lib.FragmentSize)
	}
	lib.Fragments = append(lib.Fragments, frag)
	return nil
}

// Save saves the full fragment library to the writer provied.
func (lib *StructureLibrary) Save(w io.Writer) error {
	enc := gob.NewEncoder(w)
	return enc.Encode(*lib)
}

// Open loads an existing structure fragment library from the reader provided.
func OpenStructureLibrary(r io.Reader) (*StructureLibrary, error) {
	var lib *StructureLibrary

	dec := gob.NewDecoder(r)
	if err := dec.Decode(&lib); err != nil {
		return nil, err
	}
	return lib, nil
}

// Size returns the number of fragments in the library.
func (lib *StructureLibrary) Size() int {
	return len(lib.Fragments)
}

// String returns a string with the name of the library, the number of
// fragments in the library and the size of each fragment.
func (lib *StructureLibrary) String() string {
	return fmt.Sprintf("%s (%d, %d)",
		lib.Ident, len(lib.Fragments), lib.FragmentSize)
}

func (lib *StructureLibrary) Name() string {
	return lib.Ident
}

// rmsdMemory creates reusable memory for use with RMSD calculation with
// suitable size for this fragment library. Only one goroutine can use the
// memory at a time.
func (lib *StructureLibrary) rmsdMemory() structure.Memory {
	return structure.NewMemory(lib.FragmentSize)
}

// Best returns the number of the fragment that best corresponds
// to the region of atoms provided.
// The length of `atoms` must be equivalent to the fragment size.
func (lib *StructureLibrary) Best(atoms []structure.Coords) int {
	return lib.bestMem(atoms, lib.rmsdMemory())
}

// BestMem returns the number of the fragment that best corresponds
// to the region of atoms provided without allocating.
// The length of `atoms` must be equivalent to the fragment size.
//
// `mem` must be a region of reusable memory that should only be accessed
// from one goroutine at a time. Valid values can be constructed with
// rmsdMemory.
func (lib *StructureLibrary) bestMem(
	atoms []structure.Coords,
	mem structure.Memory,
) int {
	var testRmsd float64
	bestRmsd, bestFragNum := 0.0, -1
	for _, frag := range lib.Fragments {
		testRmsd = structure.RMSDMem(mem, atoms, frag.Atoms)
		if bestFragNum == -1 || testRmsd < bestRmsd {
			bestRmsd, bestFragNum = testRmsd, frag.Number
		}
	}
	return bestFragNum
}

// Fragment corresponds to a single structural fragment in a fragment library.
// It holds the fragment number identifier and the 3 dimensional coordinates.
type StructureFragment struct {
	Number int
	Atoms  []structure.Coords
}

func (frag *StructureFragment) FragNumber() int {
	return frag.Number
}

// String returns the fragment number, library and its corresponding atoms.
func (frag *StructureFragment) String() string {
	atoms := make([]string, len(frag.Atoms))
	for i, atom := range frag.Atoms {
		atoms[i] = fmt.Sprintf("\t%s", atom)
	}
	return fmt.Sprintf("> %d\n%s", frag.Number, strings.Join(atoms, "\n"))
}
