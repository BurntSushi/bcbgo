package seq

import (
	"fmt"
	"strings"
)

type MSA struct {
	Entries []Sequence
	length  int
}

func NewMSA() MSA {
	return MSA{
		Entries: make([]Sequence, 0, 5),
		length:  0,
	}
}

// Len returns the length of the alignment. (All entries in an MSA are
// guaranteed to have the same length.)
func (m MSA) Len() int {
	return m.length
}

// Slice returns a slice of the given MSA by slicing each sequence in the MSA.
// Slicing an empty MSA will panic.
//
// Note that slicing works in terms of *match states* of the first sequence
// in the MSA.
func (m MSA) Slice(start, end int) MSA {
	if len(m.Entries) == 0 {
		panic("Cannot slice an empty MSA.")
	}

	// We need to find the real start and end based on match states
	// in the first sequence.
	// Remember, entries are stored in A2M format.
	colStart, colEnd := 0, -1
	for i, residue := range m.Entries[0].Residues {
		if colStart == start {
			start = i
			break
		}
		if residue.HMMState() != Insertion {
			colStart += 1
		}
	}
	for i, residue := range m.Entries[0].Residues {
		if residue.HMMState() != Insertion {
			colEnd += 1
		}
		if colEnd == end {
			end = i
			break
		}
		if i == len(m.Entries[0].Residues)-1 {
			end = len(m.Entries[0].Residues)
		}
	}

	entries := make([]Sequence, len(m.Entries))
	for i, entry := range m.Entries {
		entries[i] = entry.Slice(start, end)
	}
	return MSA{
		Entries: entries,
		length:  end - start,
	}
}

// AddSlice calls "Add" for each sequence in the slice.
func (m *MSA) AddSlice(seqs []Sequence) {
	for _, s := range seqs {
		m.Add(s)
	}
}

// Add adds a new entry to the multiple sequence alignment. Sequences must be in
// FASTA, A2M or A3M format.
//
// Empty sequences are ignored.
func (m *MSA) Add(adds Sequence) {
	if adds.Len() == 0 {
		return
	}

	// We will, in all likelihood, modify the sequence if it's in A3M format.
	// So we copy it to prevent weird effects to the caller.
	s := adds.Copy()

	// The first sequence is easy.
	if m.length == 0 {
		m.Entries = append(m.Entries, s)
		m.length = s.Len()
		return
	}

	// Things are much easier when the sequence we're adding already has the
	// same length as the alignment. All we need to do is check FASTA formats
	// and replace '-' with '.' in insertion columns.
	if m.length == s.Len() {
		m.Entries = append(m.Entries, s)
		for col := 0; col < m.length; col++ {
			if m.columnHasInsertion(col) {
				for _, other := range m.Entries {
					if other.Residues[col] == '-' {
						other.Residues[col] = '.'
					}
				}
			}
		}
		return
	}

	// This should be an A3M formatted sequence (no way to do a sanity check
	// though, I don't think).
	// Therefore, we need to assimilate it with other sequences, which will
	// change the length of the MSA. In particular, we'll need to add '.'
	// residues to existing sequences.
	for col := 0; col < m.length; col++ {
		colHasInsert := m.columnHasInsertion(col)
		if col >= s.Len() {
			if colHasInsert {
				s.Residues = append(s.Residues, '.')
			} else {
				s.Residues = append(s.Residues, '-')
			}
			continue
		}

		seqHasInsert := s.Residues[col].HMMState() == Insertion
		switch {
		case colHasInsert && seqHasInsert:
			// do nothing, we're in sync
		case colHasInsert:
			// Put an insert into the sequence we're adding.
			addInsert(&s.Residues, col)
		case seqHasInsert:
			// Put an insert into the rest of the sequences.
			for i := range m.Entries {
				addInsert(&m.Entries[i].Residues, col)
			}
			m.length++
		default:
			// This is a match/delete column, so we're good.
		}
	}
	m.Entries = append(m.Entries, s)
}

func addInsert(rs *[]Residue, col int) {
	*rs = append((*rs)[:col], append([]Residue{'.'}, (*rs)[col:]...)...)
}

func (m MSA) columnHasInsertion(col int) bool {
	for _, s := range m.Entries {
		if s.Residues[col].HMMState() == Insertion {
			return true
		}
	}
	return false
}

// Get gets a copy of the sequence at the provided row in the MSA.
// The sequence is in the default format of the MSA representation. Currently,
// this is A2M. ('-' for deletes, '.' and a-z for inserts, and A-Z for matches.)
func (m MSA) Get(row int) Sequence {
	return m.GetA2M(row)
}

// GetFasta gets a copy of the sequence at the provided row in the MSA in
// aligned FASTA format.
// This is the same as A2M format, except all '.' (period) are changed to '-'
// residues.
func (m MSA) GetFasta(row int) Sequence {
	s := m.Entries[row].Copy()
	for i, r := range s.Residues {
		if r == '.' {
			s.Residues[i] = '-'
		}
	}
	return s
}

// GetA2M gets a copy of the sequence at the provided row in the MSA in A2M
// format.
func (m MSA) GetA2M(row int) Sequence {
	return m.Entries[row].Copy()
}

// GetA3M gets a copy of the sequence at the provided row in the MSA in A3M
// format.
// This is the same as A2M format, except all '.' (period) are omitted.
func (m MSA) GetA3M(row int) Sequence {
	s := m.Entries[row]
	residues := make([]Residue, 0, s.Len())
	for _, r := range s.Residues {
		if r != '.' {
			residues = append(residues, r)
		}
	}
	return Sequence{
		Name:     fmt.Sprintf("%s", s.Name),
		Residues: residues,
	}
}

func (m MSA) String() string {
	entries := make([]string, len(m.Entries))
	for i, s := range m.Entries {
		entries[i] = fmt.Sprintf(">%s\n%s", s.Name, s.Residues)
	}
	return strings.Join(entries, "\n")
}
