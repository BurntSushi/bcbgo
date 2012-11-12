package seq

type HMMState int

// HMM states in the Plan7 architecture.
const (
	Match HMMState = iota
	Deletion
	Insertion
	Begin
	End
)
