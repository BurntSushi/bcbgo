package fragbag

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
	library   *Library
	diffFreqs []int16
}

// NewBowDiff creates a new BOWDiff by subtracting the 'old' frequencies from
// the 'new' frequencies.
//
// NewBowDiff will panic if 'oldbow' and 'newbow' weren't generated from the
// same library.
func NewBowDiff(oldbow, newbow BOW) BOWDiff {
	mustHaveSameLibrary(oldbow, newbow)
	mustHaveSameLength(oldbow, newbow)

	dfreqs := make([]int16, len(oldbow.fragfreqs))
	for i := range oldbow.fragfreqs {
		oldfreq := oldbow.fragfreqs[i]
		newfreq := newbow.fragfreqs[i]
		dfreqs[i] = newfreq - oldfreq
	}
	return BOWDiff{
		library:   oldbow.library,
		diffFreqs: dfreqs,
	}
}

// IsSame returns true if there are no differences. (i.e., all diff frequencies
// are zero.)
func (bdiff BOWDiff) IsSame() bool {
	for _, dfreq := range bdiff.diffFreqs {
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
	for i := 0; i < bdiff.library.Size(); i++ {
		freq := bdiff.diffFreqs[i]
		if freq != 0 {
			pieces = append(pieces, fmt.Sprintf("%d: %d", i, freq))
		}
	}
	return fmt.Sprintf("{%s}", strings.Join(pieces, ", "))
}
