package seq

import (
	"fmt"
	"math"
	"strconv"
)

// HMM states in the Plan7 architecture.
const (
	Match HMMState = iota
	Deletion
	Insertion
	Begin
	End
)

type HMMState int

type HMM struct {
	// An ordered list of HMM nodes.
	Nodes []HMMNode

	// The alphabet as defined by an ordering of residues.
	// Indices in this slice correspond to indices in match/insertion emissions.
	Alphabet []Residue

	// NULL model. (Amino acid background frequencies.)
	// HMMER hmm files don't have this, but HHsuite hhm files do.
	// In the case of HHsuite, the NULL model is used for insertion emissions
	// in every node.
	Null EProbs
}

type HMMNode struct {
	HMM                 *HMM
	Residue             Residue
	NodeNum             int
	InsEmit             EProbs
	MatEmit             EProbs
	Transitions         TProbs
	NeffM, NeffI, NeffD Prob
}

// EProbs represents emission probabilities, as log-odds scores.
type EProbs map[Residue]Prob

// NewEProbs creates a new EProbs map from the given alphabet. Keys of the map
// are residues defined in the alphabet, and values are defaulted to the
// minimum probability.
func NewEProbs(alphabet []Residue) EProbs {
	ep := make(EProbs, len(alphabet))
	for _, residue := range alphabet {
		ep[residue] = MinProb
	}
	return ep
}

// Returns the emission probability for a particular residue.
func (ep EProbs) EmitProb(r Residue) Prob {
	return ep[r]
}

// TProbs represents transition probabilities, as log-odds scores.
// Note that ID and DI are omitted (Plan7).
type TProbs struct {
	MM, MI, MD, IM, II, DM, DD Prob
}

// Prob represents a transition or emission probability.
type Prob float64

var invalidProb = Prob(math.NaN())

// The value representing a minimum emission/transition probability.
// Remember, max in log space is minimum probability.
var MinProb = Prob(math.MaxFloat64)

// NewProb creates a new probability value from a string (usually read from
// an hmm or hhm file). If the string is equivalent to the special value "*",
// then the probability returned is guaranteed to be minimal. Otherwise, the
// string is parsed as a float, and an error returned if parsing fails.
func NewProb(fstr string) (Prob, error) {
	if fstr == "*" {
		return MinProb, nil
	}

	f, err := strconv.ParseFloat(fstr, 64)
	if err != nil {
		return invalidProb,
			fmt.Errorf("Could not convert '%s' to a log probability: %s",
				fstr, err)
	}
	return Prob(f), nil
}

// IsMin returns true if the probability is minimal.
func (p Prob) IsMin() bool {
	return p == MinProb
}

// NewHMM creates a new HMM from a list of nodes, an ordered alphabet and a
// set of null probabilities (which may be nil).
func NewHMM(nodes []HMMNode, alphabet []Residue, null EProbs) *HMM {
	return &HMM{
		Nodes:    nodes,
		Alphabet: alphabet,
		Null:     null,
	}
}
