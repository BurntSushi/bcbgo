package seq

type Alphabet []Residue

func NewAlphabet(residues ...Residue) Alphabet {
	return Alphabet(residues)
}

func (a Alphabet) Len() int {
	return len(a)
}

const alpha62letters = "ABCDEFGHIKLMNPQRSTVWXYZ-"

// The default alphabet that corresponds to the BLOSUM62 matrix included
// in this package.
var AlphaBlosum62 = NewAlphabet(
	'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'K', 'L', 'M',
	'N', 'P', 'Q', 'R', 'S', 'T', 'V', 'W', 'X', 'Y', 'Z', '-',
)
