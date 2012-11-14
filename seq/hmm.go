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

	// alphaIndices is the reverse of Alphabet. It lets us access emissions
	// for a given residue in constant time. Classic time/space trade off.
	alphaIndices map[Residue]int

	// NULL model. (Amino acid background frequencies.)
	// HMMER hmm files don't have this, but HHsuite hhm files do.
	// In the case of HHsuite, the NULL model is used for insertion emissions
	// in every node.
	Null EProbs
}

type HMMNode struct {
	HMM                 *HMM
	NodeNum             int
	Transitions         TProbs
	InsEmit             EProbs
	MatEmit             EProbs
	NeffM, NeffI, NeffD Prob
}

// EProbs represents emission probabilities, as log-odds scores.
type EProbs []Prob

func (ep EProbs) EmitProb(hmm *HMM, r Residue) Prob {
	return ep[hmm.alphaIndices[r]]
}

// TProbs represents transition probabilities, as log-odds scores.
// Note that ID and DI are omitted (Plan7).
type TProbs struct {
	MM, MI, MD, IM, II, DM, DD Prob
}

// Prob represents a transition or emission probability.
type Prob float64

var invalidProb = Prob(math.NaN())

var minProb = Prob(math.MaxFloat64) // max in log space is minimum probability

func NewProb(fstr string) (Prob, error) {
	if fstr == "*" {
		return minProb, nil
	}

	f, err := strconv.ParseFloat(fstr, 64)
	if err != nil {
		return invalidProb,
			fmt.Errorf("Could not convert '%s' to a log probability: %s",
				fstr, err)
	}
	return Prob(f), nil
}

func (p Prob) IsValid() bool {
	return p != invalidProb
}

func (p Prob) IsMin() bool {
	return p == minProb
}

func NewHMM(nodes []HMMNode, alphabet []Residue, null EProbs) *HMM {
	hmm := &HMM{
		Nodes:        nodes,
		Alphabet:     alphabet,
		alphaIndices: make(map[Residue]int, len(alphabet)),
		Null:         null,
	}
	for index, residue := range alphabet {
		hmm.alphaIndices[residue] = index
	}
	return hmm
}

func IsMaxFloat64(f float64) bool {
	return f == math.MaxFloat64
}
