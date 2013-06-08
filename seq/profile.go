package seq

import (
	"fmt"
	"math"
)

// Profile represents a sequence profile in terms of log-odds scores.
type Profile struct {
	// The columns of a profile.
	Emissions []EProbs

	// The alphabet of the profile. The length of the alphabet should be
	// equal to the number of rows in the profile.
	// There are no restrictions on the alphabet. (i.e., Gap characters are
	// allowed but they are not treated specially.)
	Alphabet Alphabet
}

// NewProfile initializes a profile with a default
// alphabet that is compatible with this package's BLOSUM62 matrix.
// Emission probabilities are set to the minimum log-odds probability.
func NewProfile(columns int) *Profile {
	return NewProfileAlphabet(columns, AlphaBlosum62)
}

// NewProfileAlphabet initializes a profile with the given alphabet.
// Emission probabilities are set to the minimum log-odds probability.
func NewProfileAlphabet(columns int, alphabet Alphabet) *Profile {
	emits := make([]EProbs, columns)
	for i := 0; i < columns; i++ {
		emits[i] = NewEProbs(alphabet)
	}
	return &Profile{emits, alphabet}
}

func (p *Profile) Len() int {
	return len(p.Emissions)
}

// FrequencyProfile represents a sequence profile in terms of raw frequencies.
// A FrequencyProfile is useful as an intermediate representation. It can be
// used to incrementally build a Profile.
type FrequencyProfile struct {
	// The columns of a frequency profile.
	Freqs []map[Residue]int

	// The alphabet of the profile. The length of the alphabet should be
	// equal to the number of rows in the frequency profile.
	// There are no restrictions on the alphabet. (i.e., Gap characters are
	// allowed but they are not treated specially.)
	Alphabet Alphabet
}

// NewFrequencyProfile initializes a raw frequency profile with a default
// alphabet that is compatible with this package's BLOSUM62 matrix.
// Pseudo-count correction using Laplace's Rule is automatically applied.
func NewFrequencyProfile(columns int) *FrequencyProfile {
	return NewFrequencyProfileAlphabet(columns, AlphaBlosum62)
}

// NewFrequencyProfileAlphabet initializes a raw frequency profile with the
// given alphabet.
// Pseudo-count correction using Laplace's Rule is automatically applied.
func NewFrequencyProfileAlphabet(
	columns int,
	alphabet Alphabet,
) *FrequencyProfile {
	freqs := make([]map[Residue]int, columns)
	for i := 0; i < columns; i++ {
		freqs[i] = make(map[Residue]int, len(alphabet))
		for _, residue := range alphabet {
			freqs[i][residue] = 1 // Laplace's rule
		}
	}
	return &FrequencyProfile{freqs, alphabet}
}

// Len returns the number of columns in the frequency profile.
func (fp *FrequencyProfile) Len() int {
	return len(fp.Freqs)
}

// Profile converts a raw frequency profile to a profile that uses a log-odds
// representation. The log-odds scores are computed with the given null model,
// which is itself just a raw frequency profile with a single column.
// Pseudo-count correction using Laplace's Rule is automatically applied.
// The alphabets of `fp` and `null` must be exactly the same.
func (fp *FrequencyProfile) Profile(null *FrequencyProfile) *Profile {
	if null.Len() != 1 {
		panic(fmt.Sprintf("null model has %d columns; should have 1",
			null.Len()))
	}
	p := NewProfileAlphabet(fp.Len(), fp.Alphabet)

	// Compute the background emission probabilities.
	nulltot := freqTotal(null.Freqs[0])
	nullemit := make(map[Residue]float64, fp.Alphabet.Len())
	for _, residue := range null.Alphabet {
		nullemit[residue] = float64(null.Freqs[0][residue]) / float64(nulltot)
	}

	// Now compute the emission probabilities and convert to log-odds.
	for column := 0; column < fp.Len(); column++ {
		tot := freqTotal(fp.Freqs[column])
		for _, residue := range fp.Alphabet {
			prob := float64(fp.Freqs[column][residue]) / float64(tot)
			logOdds := Prob(math.Log(prob / nullemit[residue]))
			p.Emissions[column][residue] = logOdds
		}
	}
	return p
}

// freqTotal computes the total frequency in a single column.
func freqTotal(column map[Residue]int) int {
	tot := 0
	for _, freq := range column {
		tot += freq
	}
	return tot
}
