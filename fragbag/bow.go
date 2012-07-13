package fragbag

import (
	"fmt"
	"strings"

	"github.com/BurntSushi/bcbgo/pdb"
)

// BOW represents a bag-of-words vector of size N for a particular fragment
// library, where N corresponds to the number of fragments in the fragment
// library.
type BOW struct {
	// library corresponds to the fragment library used to compute this
	// bag-of-words. All BOW operations are only defined when all operands 
	// belong to the same fragment library.
	library *Library

	// fragfreqs is a map from fragment number to the number of occurrences of
	// that fragment in this "bag of words." This map always has size
	// equivalent to the size of the library.
	fragfreqs []int16
}

// NewBow returns a bag-of-words with all fragment frequencies set to 0.
func (lib *Library) NewBow() BOW {
	bow := BOW{
		library:   lib,
		fragfreqs: make([]int16, lib.Size()),
	}
	for i := range lib.fragments {
		bow.fragfreqs[i] = 0
	}
	return bow
}

// NewPDB returns a bag-of-words describing a pdb file.
//
// All protein chains in the PDB file are used.
func (lib *Library) NewBowPDB(entry *pdb.Entry) BOW {
	// We don't use the public 'Increment' or 'Add' methods to avoid
	// excessive allocations.
	bow := lib.NewBow()

	// Create a list of atom sets for all K-mer windows of all protein chains
	// in the PDB entry, where K is the fragment size of the library.
	// The list of atom sets can then have the best fragment for each atom
	// set computed concurrently with BestFragments.
	atomSets := make([]pdb.Atoms, 0, 100)

	for _, chain := range entry.Chains {
		if !chain.ValidProtein() {
			continue
		}

		// If this chain is smaller than the fragment size, then we skip it.
		if len(chain.CaAtoms) < lib.FragmentSize() {
			continue
		}

		// Otherwise, the chain is bigger than the fragment size. So add each
		// of its K-mer windows to the atom set.
		for i := 0; i <= len(chain.CaAtoms)-lib.FragmentSize(); i++ {
			atomSets = append(atomSets, chain.CaAtoms[i:i+lib.FragmentSize()])
		}
	}

	// Get the best fragment numbers for each set, and increase the frequency
	// of each fragment number returned.
	for _, bestFragNum := range lib.BestFragments(atomSets) {
		if bestFragNum >= 0 {
			bow.fragfreqs[bestFragNum] += 1
		}
	}
	return bow
}

// Increment will increment the frequency of the given fragment number by 1.
// If a fragment of fragNum does not exist, Increment will panic.
func (bow BOW) Increment(fragNum int) {
	bow.library.mustExist(fragNum)
	bow.fragfreqs[fragNum] += 1
}

// Add performs an add operation on each fragment frequency and returns
// a new BOW. Add will panic if the operands came from different fragment
// libraries.
func (bow1 BOW) Add(bow2 BOW) BOW {
	mustHaveSameLibrary(bow1, bow2)

	sum := bow1.library.NewBow()
	for i := 0; i < sum.library.Size(); i++ {
		sum.fragfreqs[i] = bow1.fragfreqs[i] + bow2.fragfreqs[i]
	}
	return sum
}

// String returns a string representation of the BOW vector. Only fragments
// with non-zero frequency are emitted.
//
// The output looks like '{fragNum: frequency, fragNum: frequency, ...}'.
// i.e., '{1: 4, 3: 1}' where all fragment numbers except '1' and '3' have
// a frequency of zero.
func (bow BOW) String() string {
	pieces := make([]string, 0, 10)
	for i := 0; i < bow.library.Size(); i++ {
		freq := bow.fragfreqs[i]
		if freq > 0 {
			pieces = append(pieces, fmt.Sprintf("%d: %d", i, freq))
		}
	}
	return fmt.Sprintf("{%s}", strings.Join(pieces, ", "))
}

// mustHaveSameLibrary panics if any bow in bows belongs to a different
// library than any other bow in bows. (Libraries are compared using pointer
// equality.)
func mustHaveSameLibrary(bows ...BOW) {
	var lib *Library = nil
	for _, bow := range bows {
		if bow.library == nil {
			panic(fmt.Sprintf("A BOW does not belong to any library."))
		}
		if lib == nil {
			lib = bow.library
			continue
		}
		if lib != bow.library {
			panic(fmt.Sprintf("A BOW belongs to library '%s', but another "+
				"BOW belongs to library '%s'.", bow.library, lib))
		}
	}
}
