package bow

import (
	"fmt"
	"math"
	"strings"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/io/pdb"
	"github.com/BurntSushi/bcbgo/rmsd"
)

// Bower should be:
//
// Id() string
// Data() string
//
// Then `StructureBower`
// AtomChunks() [][]pdb.Coords
//
// And `SeqBower`
// Residues() []seq.Residue
//
// Move to Fragbag package.
// Split Library into Structure and Sequence libraries.
//
// Similarly for DB in `bow` package. Code reuse through embedding.

type Bower interface {
	IdString() string
	AtomChunks() [][]pdb.Coords
}

func ComputeBOWMem(
	lib *fragbag.Library, bower Bower, mem rmsd.QcMemory) BOW {

	b := NewBow(lib.Size())
	libSize := lib.FragmentSize()
	frags := lib.Fragments()
	for _, chunk := range bower.AtomChunks() {
		if len(chunk) < libSize {
			continue
		}
		for i := 0; i <= len(chunk)-libSize; i++ {
			atoms := chunk[i : i+libSize]
			bestFragNum := bestFragment(frags, mem, atoms)
			b.Freqs[bestFragNum] += 1
		}
	}
	return b
}

func ComputeBOW(lib *fragbag.Library, bower Bower) BOW {
	mem := rmsd.NewQcMemory(lib.FragmentSize())
	return ComputeBOWMem(lib, bower, mem)
}

func bestFragment(frags []fragbag.Fragment,
	mem rmsd.QcMemory, atoms []pdb.Coords) int {

	bestRmsd, bestFragNum := 0.0, -1
	for _, frag := range frags {
		testRmsd := rmsd.QCRMSDMem(mem, atoms, frag.CaAtoms)
		if bestFragNum == -1 || testRmsd < bestRmsd {
			bestRmsd, bestFragNum = testRmsd, frag.Ident
		}
	}
	return bestFragNum
}

// BOW represents a bag-of-words vector of size N for a particular fragment
// library, where N corresponds to the number of fragments in the fragment
// library.
type BOW struct {
	// Freqs is a map from fragment number to the number of occurrences of
	// that fragment in this "bag of words." This map always has size
	// equivalent to the size of the library.
	Freqs []uint32
}

// NewBow returns a bag-of-words with all fragment frequencies set to 0.
func NewBow(size int) BOW {
	bow := BOW{
		Freqs: make([]uint32, size),
	}
	for i := 0; i < size; i++ {
		bow.Freqs[i] = 0
	}
	return bow
}

// Len returns the size of the vector. This is always equivalent to the
// corresponding library's fragment size.
func (bow BOW) Len() int {
	return len(bow.Freqs)
}

// Equal tests whether two BOWs are equal.
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
// a new BOW. Add will panic if the operands have different lengths.
func (bow1 BOW) Add(bow2 BOW) BOW {
	if bow1.Len() != bow2.Len() {
		panic("Cannot add two BOWs with differing lengths")
	}

	sum := NewBow(bow1.Len())
	for i := 0; i < sum.Len(); i++ {
		sum.Freqs[i] = bow1.Freqs[i] + bow2.Freqs[i]
	}
	return sum
}

// Euclid returns the euclidean distance between bow1 and bow2.
func (bow1 BOW) Euclid(bow2 BOW) float64 {
	f1, f2 := bow1.Freqs, bow2.Freqs
	squareSum := uint32(0)
	libsize := bow1.Len()
	for i := 0; i < libsize; i++ {
		squareSum += (f2[i] - f1[i]) * (f2[i] - f1[i])
	}
	return math.Sqrt(float64(squareSum))
}

// Cosine returns the cosine distance between bow1 and bow2.
func (bow1 BOW) Cosine(bow2 BOW) float64 {
	// This function is a hot-spot, so we manually inline the Dot
	// and Magnitude computations.

	var dot, mag1, mag2 uint32
	libs := len(bow1.Freqs)
	freqs1, freqs2 := bow1.Freqs, bow2.Freqs

	var f1, f2 uint32
	for i := 0; i < libs; i++ {
		f1, f2 = freqs1[i], freqs2[i]
		dot += f1 * f2
		mag1 += f1 * f1
		mag2 += f2 * f2
	}
	r := 1.0 - (float64(dot) / math.Sqrt(float64(mag1)*float64(mag2)))
	if math.IsNaN(r) {
		return 1.0
	}
	return r
}

// Dot returns the dot product of bow1 and bow2.
func (bow1 BOW) Dot(bow2 BOW) float64 {
	dot := uint32(0)
	libsize := bow1.Len()
	f1, f2 := bow1.Freqs, bow2.Freqs
	for i := 0; i < libsize; i++ {
		dot += f1[i] * f2[i]
	}
	return float64(dot)
}

// Magnitude returns the vector length of the bow.
func (bow BOW) Magnitude() float64 {
	mag := uint32(0)
	libsize := bow.Len()
	fs := bow.Freqs
	for i := 0; i < libsize; i++ {
		mag += fs[i] * fs[i]
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
