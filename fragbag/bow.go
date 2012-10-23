package fragbag

import (
	"fmt"
	"math"
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

// NewBowMap returns a bag-of-words with the vector initialized to the map
// provided. The keys of the map should be fragment numbers and the values
// should be frequencies.
func (lib *Library) NewBowMap(freqMap map[int]int16) BOW {
	bow := lib.NewBow()
	for fragNum, freq := range freqMap {
		bow.fragfreqs[fragNum] = freq
	}
	return bow
}

// NewBowChain returns a bag-of-words describing a chain in a PDB entry.
func (lib *Library) NewBowChain(chain *pdb.Chain) BOW {
	bow := lib.NewBow()
	if len(chain.CaAtoms) < lib.FragmentSize() {
		return bow
	}
	for i := 0; i <= len(chain.CaAtoms)-lib.FragmentSize(); i++ {
		atoms := chain.CaAtoms[i : i+lib.FragmentSize()]
		bestFragNum, _ := lib.BestFragment(atoms)
		bow.fragfreqs[bestFragNum]++
	}
	return bow
}

// NewBowPDB returns a bag-of-words describing a PDB file without concurrency.
// This is useful when computing the BOW of many PDB files, and the level
// of concurrency should be at the level of computing BOWs rather than
// RMSDs for each fragment.
func (lib *Library) NewBowPDB(entry *pdb.Entry) BOW {
	bow := lib.NewBow()
	for _, chain := range entry.Chains {
		if !chain.ValidProtein() {
			continue
		}
		if len(chain.CaAtoms) < lib.FragmentSize() {
			continue
		}
		for i := 0; i <= len(chain.CaAtoms)-lib.FragmentSize(); i++ {
			atoms := chain.CaAtoms[i : i+lib.FragmentSize()]
			bestFragNum, _ := lib.BestFragment(atoms)
			bow.fragfreqs[bestFragNum]++
		}
	}
	return bow
}

// NewBowPDBPar returns a bag-of-words describing a PDB file by computing
// the RMSD of each fragment in the PDB file concurrently.
//
// All protein chains in the PDB file are used.
func (lib *Library) NewBowPDBPar(entry *pdb.Entry) BOW {
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

// Frequency returns the number of times the fragment numbered fragNum appears
// in the BOW vector.
func (bow BOW) Frequency(fragNum int) int16 {
	bow.library.mustExist(fragNum)
	return bow.fragfreqs[fragNum]
}

// Len returns the size of the vector. This is always equivalent to the
// corresponding library's fragment size.
func (bow BOW) Len() int {
	return len(bow.fragfreqs)
}

// Equal tests whether two fragments are equal. In order for "equality" to
// be defined, both fragments MUST be from the same library. If they aren't,
// Equal will panic.
//
// Two BOWs are equivalent when the frequencies of every fragment are equal.
func (bow1 BOW) Equal(bow2 BOW) bool {
	mustHaveSameLibrary(bow1, bow2)
	mustHaveSameLength(bow1, bow2)
	for i, freq1 := range bow1.fragfreqs {
		if freq1 != bow2.fragfreqs[i] {
			return false
		}
	}
	return true
}

// Add performs an add operation on each fragment frequency and returns
// a new BOW. Add will panic if the operands came from different fragment
// libraries.
func (bow1 BOW) Add(bow2 BOW) BOW {
	mustHaveSameLibrary(bow1, bow2)
	mustHaveSameLength(bow1, bow2)

	sum := bow1.library.NewBow()
	for i := 0; i < sum.library.Size(); i++ {
		sum.fragfreqs[i] = bow1.fragfreqs[i] + bow2.fragfreqs[i]
	}
	return sum
}

// Euclid returns the euclidean distance between bow1 and bow2.
func (bow1 BOW) Euclid(bow2 BOW) float64 {
	f1, f2 := bow1.fragfreqs, bow2.fragfreqs
	squareSum := float64(0)
	for i := 0; i < bow1.library.Size(); i++ {
		squareSum += float64(int32(f2[i]-f1[i]) * int32(f2[i]-f1[i]))
	}
	return math.Sqrt(squareSum)
}

// Cosine returns the cosine distance between bow1 and bow2.
func (bow1 BOW) Cosine(bow2 BOW) float64 {
	r := 1.0 - (bow1.Dot(bow2) / (bow1.Magnitude() * bow2.Magnitude()))
	if math.IsNaN(r) {
		return 1.0
	}
	return r
}

// Dot returns the dot product of bow1 and bow2.
func (bow1 BOW) Dot(bow2 BOW) float64 {
	dot := float64(0)
	for i := 0; i < bow1.library.Size(); i++ {
		dot += float64(int32(bow1.fragfreqs[i]) * int32(bow2.fragfreqs[i]))
	}
	return dot
}

// magnitude returns the vector length of the bow.
func (bow BOW) Magnitude() float64 {
	mag := float64(0)
	for i := 0; i < bow.library.Size(); i++ {
		mag += float64(int32(bow.fragfreqs[i]) * int32(bow.fragfreqs[i]))
	}
	return math.Sqrt(mag)
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

// mustHaveSameLength panics if any two BOWs have differing lengths when they
// were expected to have the same. (i.e., it is appropriate to call this
// right after 'mustHaveSameLibrary', but NOT before.)
//
// This exists to discover bugs.
func mustHaveSameLength(bows ...BOW) {
	lenMatch, refBow := -1, BOW{}
	for _, bow := range bows {
		if lenMatch == -1 {
			lenMatch = len(bow.fragfreqs)
			refBow = bow
			continue
		}
		if lenMatch != len(bow.fragfreqs) {
			panic(fmt.Sprintf("BUG: Two BOWs belonging to the same library "+
				"have lengths %d and %d. The BOWs are \n\n%s\n\n%s.\n",
				lenMatch, len(bow.fragfreqs), refBow, bow))
		}
	}
}
