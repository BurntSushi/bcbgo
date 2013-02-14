package bow

import (
	"fmt"
	"strings"
)

// BOWDiff represents the difference between two bag-of-words vectors. The
// types are quite similar, except diffFreqs represents difference between
// the frequency of a particular fragment number.
//
// The BOW difference is simply the pairwise differences of fragment
// frequencies.
type BOWDiff struct {
	Freqs []int32
}

// NewBowDiff creates a new BOWDiff by subtracting the 'old' frequencies from
// the 'new' frequencies.
//
// NewBowDiff will panic if 'oldbow' and 'newbow' weren't generated from the
// same library.
func NewBowDiff(oldbow, newbow BOW) BOWDiff {
	if len(oldbow.Freqs) != len(newbow.Freqs) {
		panic("Cannot diff two BOWs with differing lengths")
	}

	dfreqs := make([]int32, len(oldbow.Freqs))
	for i := range oldbow.Freqs {
		oldfreq := oldbow.Freqs[i]
		newfreq := newbow.Freqs[i]
		dfreqs[i] = int32(newfreq) - int32(oldfreq)
	}
	return BOWDiff{
		Freqs: dfreqs,
	}
}

// IsSame returns true if there are no differences. (i.e., all diff frequencies
// are zero.)
func (bdiff BOWDiff) IsSame() bool {
	for _, dfreq := range bdiff.Freqs {
		if dfreq != 0 {
			return false
		}
	}
	return true
}

// String returns a string representation of the BOW diff vector. Only fragments
// with non-zero differences are emitted.
//
// The output looks like
// '{fragNum: diff-frequency, fragNum: diff-frequency, ...}'.
// i.e., '{1: 4, 3: 1}' where all fragment numbers except '1' and '3' have
// a difference frequency of zero.
func (bdiff BOWDiff) String() string {
	pieces := make([]string, 0, 10)
	for i := 0; i < len(bdiff.Freqs); i++ {
		freq := bdiff.Freqs[i]
		if freq != 0 {
			pieces = append(pieces, fmt.Sprintf("%d: %d", i, freq))
		}
	}
	return fmt.Sprintf("{%s}", strings.Join(pieces, ", "))
}
