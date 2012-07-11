package pdb

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"path"
	"os"
	"sort"
	"strconv"
	"strings"
)

var AminoThreeToOne = map[string]byte{
	"ALA": 'A', "ARG": 'R', "ASN": 'N', "ASP": 'D', "CYS": 'C',
	"GLU": 'E', "GLN": 'Q', "GLY": 'G', "HIS": 'H', "ILE": 'I',
	"LEU": 'L', "LYS": 'K', "MET": 'M', "PHE": 'F', "PRO": 'P',
	"SER": 'S', "THR": 'T', "TRP": 'W', "TYR": 'Y', "VAL": 'V',
	"SEC": 'U', "PYL": 'O',
}

var AminoOneToThree = map[byte]string{}

func init() {
	for k, v := range AminoThreeToOne {
		AminoOneToThree[v] = k
	}
}

type Entry struct {
	Chains map[byte]*Chain
}

// New creates a new PDB Entry from a file. If the file cannot be read, or there
// is an error parsing the PDB file, an error is returned.
//
// If the file name ends with ".gz", gzip decompression will be used.
func New(fileName string) (*Entry, error) {
	var reader io.Reader
	var err error

	reader, err = os.Open(fileName)
	if err != nil {
		return nil, err
	}

	// If the file is gzipped, use the gzip decompressor.
	if path.Ext(fileName) == ".gz" {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
	}

	entry := &Entry{
		Chains: make(map[byte]*Chain, 0),
	}

	// Now traverse each line, and process it according to the record name.
	breader := bufio.NewReaderSize(reader, 1000)
	for {
		// We ignore 'isPrefix' here, since we never care about lines longer
		// than 1000 characters, which is the size of our buffer.
		line, _, err := breader.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		// The record name is always in the fix six columns.
		switch strings.TrimSpace(string(line[0:6])) {
		case "SEQRES":
			entry.parseSeqres(line)
		case "ATOM":
			entry.parseAtom(line)
		}
	}

	return entry, nil
}

func (e *Entry) String() string {
	lines := make([]string, 0)
	for _, chain := range e.Chains {
		lines = append(lines, chain.String())
	}
	sort.Sort(sort.StringSlice(lines))
	return strings.Join(lines, "\n")
}

func (e *Entry) getOrMakeChain(ident byte) *Chain {
	if chain, ok := e.Chains[ident]; ok {
		return chain
	}
	e.Chains[ident] = &Chain{
		Ident: ident,
		Sequence: make([]byte, 0, 10),
		AtomResidueStart: 0,
		AtomResidueEnd: 0,
	}
	return e.Chains[ident]
}

func (e *Entry) parseSeqres(line []byte) {
	chain := e.getOrMakeChain(line[11])

	// Residues are in columns 19-21, 23-25, 27-29, ..., 67-69
	for i := 19; i <= 67; i += 4 {
		end := i+3

		// If we're passed the end of this line, quit.
		if end >= len(line) {
			break
		}

		// Get the residue. If it's not in our sequence map, skip it.
		residue := strings.TrimSpace(string(line[i:end]))
		if single, ok := AminoThreeToOne[residue]; ok {
			chain.Sequence = append(chain.Sequence, single)
		}
	}
}

func (e *Entry) parseAtom(line []byte) {
	chain := e.getOrMakeChain(line[21])

	// An ATOM record is only processed if it corresponds to an amino acid
	// residue. (Which is in columns 17-19.)
	residue := strings.TrimSpace(string(line[17:20]))
	if _, ok := AminoThreeToOne[residue]; !ok {
		return
	}

	// The residue sequence number is in columns 22-25. Grab it, trim it,
	// and look for an integer.
	snum := strings.TrimSpace(string(line[22:26]))
	if num, err := strconv.ParseInt(snum, 10, 32); err == nil {
		inum := int(num)
		switch {
		case chain.AtomResidueStart == 0 || inum < chain.AtomResidueStart:
			chain.AtomResidueStart = inum
		case chain.AtomResidueEnd == 0 || inum > chain.AtomResidueEnd:
			chain.AtomResidueEnd = inum
		}
	}
}

type Chain struct {
	Ident byte
	Sequence []byte
	AtomResidueStart, AtomResidueEnd int
}

func (c *Chain) String() string {
	return fmt.Sprintf("> Chain %c (%d, %d) :: length %d\n%s",
		c.Ident, c.AtomResidueStart, c.AtomResidueEnd,
		len(c.Sequence), string(c.Sequence))
}
