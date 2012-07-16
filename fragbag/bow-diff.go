package fragbag

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
