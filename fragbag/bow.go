package fragbag

import (
	"fmt"
	"math"
	"strings"

	"github.com/BurntSushi/bcbgo/io/pdb"
	"github.com/BurntSushi/bcbgo/rmsd"
)

// BOW represents a bag-of-words vector of size N for a particular fragment
// library, where N corresponds to the number of fragments in the fragment
// library.
type BOW struct {
	// LibraryName is an identifying string indicating how the BOW was
	// created. It is not used for anything other than diagnostic/display
	// purposes.
	LibraryName string

	// Freqs is a map from fragment number to the number of occurrences of
	// that fragment in this "bag of words." This map always has size
	// equivalent to the size of the library.
	Freqs []int16
}

// NewBow returns a bag-of-words with all fragment frequencies set to 0.
func NewBow(libraryName string, size int) BOW {
	bow := BOW{
		LibraryName: libraryName,
		Freqs:       make([]int16, size),
	}
	for i := 0; i < size; i++ {
		bow.Freqs[i] = 0
	}
	return bow
}

// NewBowMap returns a bag-of-words with the vector initialized to the map
// provided. The keys of the map should be fragment numbers and the values
// should be frequencies.
func NewBowMap(libraryName string, size int, freqMap map[int]int16) BOW {
	bow := NewBow(libraryName, size)
	for fragNum, freq := range freqMap {
		bow.Freqs[fragNum] = freq
	}
	return bow
}

// NewBowChain returns a bag-of-words describing a chain in a PDB entry.
func (lib *Library) NewBowChain(chain *pdb.Chain) BOW {
	bow := NewBow(lib.Name(), lib.Size())
	cas := chain.CaAtoms()
	if len(cas) < lib.FragmentSize() {
		return bow
	}

	mem := rmsd.NewQcMemory(lib.FragmentSize())
	for i := 0; i <= len(cas)-lib.FragmentSize(); i++ {
		atoms := cas[i : i+lib.FragmentSize()]
		bestFragNum, _ := lib.BestFragment(atoms, mem)
		bow.Freqs[bestFragNum]++
	}
	return bow
}

// NewBowPDB returns a bag-of-words describing a PDB file without concurrency.
// This is useful when computing the BOW of many PDB files, and the level
// of concurrency should be at the level of computing BOWs rather than
// RMSDs for each fragment.
func (lib *Library) NewBowPDB(entry *pdb.Entry) BOW {
	bow := NewBow(lib.Name(), lib.Size())
	mem := rmsd.NewQcMemory(lib.FragmentSize())
	for _, chain := range entry.Chains {
		if !chain.IsProtein() {
			continue
		}

		cas := chain.CaAtoms()
		if len(cas) < lib.FragmentSize() {
			continue
		}
		for i := 0; i <= len(cas)-lib.FragmentSize(); i++ {
			atoms := cas[i : i+lib.FragmentSize()]
			bestFragNum, _ := lib.BestFragment(atoms, mem)
			bow.Freqs[bestFragNum]++
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
	bow := NewBow(lib.Name(), lib.Size())

	// Create a list of atom sets for all K-mer windows of all protein chains
	// in the PDB entry, where K is the fragment size of the library.
	// The list of atom sets can then have the best fragment for each atom
	// set computed concurrently with BestFragments.
	atomSets := make([][]pdb.Coords, 0, 100)

	for _, chain := range entry.Chains {
		if !chain.IsProtein() {
			continue
		}

		cas := chain.CaAtoms()

		// If this chain is smaller than the fragment size, then we skip it.
		if len(cas) < lib.FragmentSize() {
			continue
		}

		// Otherwise, the chain is bigger than the fragment size. So add each
		// of its K-mer windows to the atom set.
		for i := 0; i <= len(cas)-lib.FragmentSize(); i++ {
			atomSets = append(atomSets, cas[i:i+lib.FragmentSize()])
		}
	}

	// Get the best fragment numbers for each set, and increase the frequency
	// of each fragment number returned.
	for _, bestFragNum := range lib.BestFragments(atomSets) {
		if bestFragNum >= 0 {
			bow.Freqs[bestFragNum] += 1
		}
	}
	return bow
}

// Len returns the size of the vector. This is always equivalent to the
// corresponding library's fragment size.
func (bow BOW) Len() int {
	return len(bow.Freqs)
}

// Equal tests whether two fragments are equal. In order for "equality" to
// be defined, both fragments MUST be from the same library. If they aren't,
// Equal will panic.
//
// Two BOWs are equivalent when the frequencies of every fragment are equal.
func (bow1 BOW) Equal(bow2 BOW) bool {
	if bow1.Len() != bow2.Len() {
		return false
	}
	for i, freq1 := range bow1.Freqs {
		if freq1 != bow2.Freqs[i] {
			return false
		}
	}
	return true
}

// Add performs an add operation on each fragment frequency and returns
// a new BOW. Add will panic if the operands came from different fragment
// libraries.
func (bow1 BOW) Add(bow2 BOW) BOW {
	if bow1.Len() != bow2.Len() {
		panic("Cannot add two BOWs with differing lengths")
	}

	sum := NewBow(bow1.LibraryName, bow1.Len())
	for i := 0; i < sum.Len(); i++ {
		sum.Freqs[i] = bow1.Freqs[i] + bow2.Freqs[i]
	}
	return sum
}

// Euclid returns the euclidean distance between bow1 and bow2.
func (bow1 BOW) Euclid(bow2 BOW) float64 {
	f1, f2 := bow1.Freqs, bow2.Freqs
	squareSum := int32(0)
	libsize := bow1.Len()
	for i := 0; i < libsize; i++ {
		squareSum += int32(f2[i]-f1[i]) * int32(f2[i]-f1[i])
	}
	return math.Sqrt(float64(squareSum))
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
	dot := int32(0)
	libsize := bow1.Len()
	for i := 0; i < libsize; i++ {
		dot += int32(bow1.Freqs[i]) * int32(bow2.Freqs[i])
	}
	return float64(dot)
}

// magnitude returns the vector length of the bow.
func (bow BOW) Magnitude() float64 {
	mag := int32(0)
	libsize := bow.Len()
	for i := 0; i < libsize; i++ {
		mag += int32(bow.Freqs[i]) * int32(bow.Freqs[i])
	}
	return math.Sqrt(float64(mag))
}

// String returns a string representation of the BOW vector. Only fragments
// with non-zero frequency are emitted.
//
// The output looks like '{fragNum: frequency, fragNum: frequency, ...}'.
// i.e., '{1: 4, 3: 1}' where all fragment numbers except '1' and '3' have
// a frequency of zero.
func (bow BOW) String() string {
	pieces := make([]string, 0, 10)
	for i := 0; i < bow.Len(); i++ {
		freq := bow.Freqs[i]
		if freq > 0 {
			pieces = append(pieces, fmt.Sprintf("%d: %d", i, freq))
		}
	}
	return fmt.Sprintf("{%s}", strings.Join(pieces, ", "))
}
