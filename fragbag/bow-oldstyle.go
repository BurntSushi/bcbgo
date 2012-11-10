package fragbag

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/BurntSushi/bcbgo/pdb"
)

// NewBowPDBOldStyle returns a bag-of-words describing a pdb file using
// a particular idiosynracy from the old Fragbag program: all carbon-alpha
// ATOM records are smushed into one slice, and each N-mer window is used
// to compute a "best" fragment. The idiosyncracy here is that chain
// information is thrown out, and RMSD computations are performed across chain
// boundaries. The regular NewBowPDB method does *not* do this.
func (lib *Library) NewBowPDBOldStyle(entry *pdb.Entry) BOW {
	// We don't use the public 'Increment' or 'Add' methods to avoid
	// excessive allocations.
	bow := lib.NewBow()

	// Flatten the atoms into one big slice first.
	atoms := make(pdb.Atoms, 0, 500)
	for _, chain := range entry.Chains {
		if !chain.ValidProtein() {
			continue
		}
		atoms = append(atoms, chain.CaAtoms...)
	}

	// Create a list of atom sets for all K-mer windows of all protein chains
	// in the PDB entry, where K is the fragment size of the library.
	// The list of atom sets can then have the best fragment for each atom
	// set computed concurrently with BestFragments.
	atomSets := make([]pdb.Atoms, 0, 100)
	for i := 0; i <= len(atoms)-lib.FragmentSize(); i++ {
		atomSets = append(atomSets, atoms[i:i+lib.FragmentSize()])
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

// This file segregates several methods the provide interoperability between
// the old Fragbag program (written by Rachel Kolodny) and this fragbag
// package. In particular, reading and writing bag-of-word vectors compatible
// with the output of the old Fragbag program.

// StringOldStyle returns a bag-of-words vector formatted as a string that
// matches the old Fragbag program's output.
//
// The format works by assigning the first 26 fragment numbers
// the letters 'a' ... 'z', the next 26 fragment numbers the letters
// 'A' ... 'Z', and any additional fragment numbers to 52, 53, 54, ..., etc.
// Moreover, the numbers are delimited by a '#' character, while the letters
// aren't delimited by anything.
//
// Here is a grammar describing the output:
//
//	output = { fragment }
//
//	fragment = lower-letter | upper-letter | { integer }, "#"
//
//	lower-letter = "a" | ... | "z"
//
//	upper-letter = "A" | ... | "Z"
//
//	integer = "0" | ... | "9"
//
// The essential invariants are that any fragment number less than 52 is
// described by elements in the set { 'a', ..., 'z', 'A', ..., 'Z' } and any
// fragment number greater than or equal to 52 is described by a corresponding
// integer (>= 52) followed by a '#' character.
//
// Note that the string returned by this function will not hold up under string
// equality with Fragbag's output. Namely, Fragbag outputs fragment numbers
// in an arbitrary order (probably the order in which they are found
// corresponding to the input PDB file). This order is not captured or
// preserved by BOW values in this package. Thus, the only way to truly test
// for equality is to convert Fragbag's output to a BOW using NewOldStyleBow,
// and using the (BOW).Equal method.
func (bow BOW) StringOldStyle() string {
	buf := new(bytes.Buffer)
	a, z := int('a'-'a'), int('z'-'a')
	A, Z := int('A'-'A'+26), int('Z'-'A'+26)
	for i, freq := range bow.fragfreqs {
		switch {
		case i >= a && i <= z:
			fragLetter := string('a' + byte(i))
			buf.WriteString(strings.Repeat(fragLetter, int(freq)))
		case i >= A && i <= Z:
			fragLetter := string('A' + byte(i) - 26)
			buf.WriteString(strings.Repeat(fragLetter, int(freq)))
		case i > Z:
			fragNum := fmt.Sprintf("%d#", i)
			buf.WriteString(strings.Repeat(fragNum, int(freq)))
		default:
			panic(fmt.Sprintf("BUG: Could not convert fragment number '%d' "+
				"to a old-style Fragbag fragment number.", i))
		}
	}
	return buf.String()
}

// NewOldStyleBow returns a bag-of-words from Fragbag's original bag-of-words
// vector output.
//
// The format works by assinging the first 26 fragment numbers
// the letters 'a' ... 'z', the next 26 fragment numbers the letters
// 'A' ... 'Z', and any additional fragment numbers to 52, 53, 54, ..., etc.
// Moreover, the numbers are delimited by a '#' character, while the letters
// aren't delimited by anything.
//
// Please see the documentation for (BOW).StringOldStyle for a production rule.
//
// If the string is malformed, NewOldStyleBow will return an error.
func (lib *Library) NewOldStyleBow(oldschool string) (BOW, error) {
	// This works by splitting the string on '#' and performing case analysis
	// on each character processed for each sub-string created by splitting
	// on '#':
	//
	//	If the character is in {'0', ..., '9'}, add the character to a byte
	//	buffer.
	//
	//	If the character is a letter in {'a', ..., 'z'}, then
	//	assign it the corresponding fragment number (ASCII Number - 'a') and
	//	increment that frequency in our BOW. Also, make sure the number buffer
	//	is empty.
	//
	//	If the character is a letter in {'A', ..., 'Z'}, then
	//	assign it the corresponding fragment number (ASCII Number - 'A' + 26)
	//	and increment that frequency in our BOW. Also, make sure the number
	//	buffer is empty.
	//
	// Each time a sub-string is processed, the contents of the buffer is
	// parsed as an integer, and the matching fragment number is increased
	// in the BOW. The buffer is subsequently emptied.
	//
	// The aforementioned exploits the fact that a '#' immediately follows
	// every fragment number that isn't represented by a letter.
	//
	// If a character not in {'0', ..., '9', 'a', ..., 'z', 'A', ..., 'Z'} is
	// found, an error is returned. If a valid character is found that doesn't
	// correspond to a valid fragment number in this library, an error is
	// returned.
	bow := lib.NewBow()
	if len(oldschool) == 0 {
		return bow, nil
	}

	mustBeEmpty := func(buf []rune, context string) error {
		if len(buf) == 0 {
			return nil
		}
		return fmt.Errorf("An unknown parse error has occurred at or around "+
			"'%s'.", context)
	}
	addToBow := func(fragNum int) error {
		if fragNum < 0 || fragNum >= len(lib.fragments) {
			return fmt.Errorf("The fragment number '%d' does not correspond "+
				"to any fragments in library %s.", fragNum, lib)
		}
		bow.Increment(fragNum)
		return nil
	}

	buf := make([]rune, 0, 15)
	for _, piece := range strings.Split(oldschool, "#") {
		for _, char := range piece {
			switch {
			case char >= '0' && char <= '9':
				buf = append(buf, char)
			case char >= 'a' && char <= 'z':
				if err := mustBeEmpty(buf, piece); err != nil {
					return BOW{}, err
				}
				if err := addToBow(int(char - 'a')); err != nil {
					return BOW{}, err
				}
			case char >= 'A' && char <= 'Z':
				if err := mustBeEmpty(buf, piece); err != nil {
					return BOW{}, err
				}
				if err := addToBow(int(char - 'A' + 26)); err != nil {
					return BOW{}, err
				}
			default:
				return BOW{}, fmt.Errorf("An unrecognized character '%c' "+
					"was found.", char)
			}
		}
		if len(buf) > 0 {
			if num64, err := strconv.ParseInt(string(buf), 10, 32); err != nil {
				return BOW{}, fmt.Errorf("Could not parse '%s' as an integer.",
					string(buf))
			} else if num64 <= 51 {
				return BOW{}, fmt.Errorf("Fragment numbers as integers must "+
					"be at least 52 or greater, but '%d' was found.", num64)
			} else {
				if err := addToBow(int(num64)); err != nil {
					return BOW{}, err
				}
			}
		}
		buf = buf[0:0]
	}
	return bow, nil
}
