package pdb

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

// AminoThreeToOne is a map from three letter amino acids to their
// corresponding single letter representation.
var AminoThreeToOne = map[string]byte{
	"ALA": 'A', "ARG": 'R', "ASN": 'N', "ASP": 'D', "CYS": 'C',
	"GLU": 'E', "GLN": 'Q', "GLY": 'G', "HIS": 'H', "ILE": 'I',
	"LEU": 'L', "LYS": 'K', "MET": 'M', "PHE": 'F', "PRO": 'P',
	"SER": 'S', "THR": 'T', "TRP": 'W', "TYR": 'Y', "VAL": 'V',
	"SEC": 'U', "PYL": 'O',
}

// AminoOneToThree is the reverse of AminoThreeToOne. It is created in
// this packages 'init' function.
var AminoOneToThree = map[byte]string{}

func init() {
	// Create a reverse map of AminoThreeToOne.
	for k, v := range AminoThreeToOne {
		AminoOneToThree[v] = k
	}
}

// Entry represents all information known about a particular PDB file (that
// has been implemented in this package).
//
// Currently, a PDB entry is simply a file path and a map of protein chains.
type Entry struct {
	Path   string
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
		Path:   fileName,
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

// String returns a sort list of all chains, their residue start/stop indics,
// and the amino acid sequence.
func (e *Entry) String() string {
	lines := make([]string, 0)
	for _, chain := range e.Chains {
		lines = append(lines, chain.String())
	}
	sort.Sort(sort.StringSlice(lines))
	return strings.Join(lines, "\n")
}

// getOrMakeChain looks for a chain in the 'Chains' map corresponding to the
// chain indentifier. If one exists, it is returned. If one doesn't exist,
// it is created, memory is allocated and it is returned.
func (e *Entry) getOrMakeChain(ident byte) *Chain {
	if chain, ok := e.Chains[ident]; ok {
		return chain
	}
	e.Chains[ident] = &Chain{
		Ident:            ident,
		Sequence:         make([]byte, 0, 10),
		AtomResidueStart: 0,
		AtomResidueEnd:   0,
	}
	return e.Chains[ident]
}

// parseSeqres loads all pertinent information from SEQRES records in a PDB
// file. In particular, amino acid resides are read and added to the chain's
// "Sequence" field. If a residue isn't a valid amino acid, it is simply
// ignored.
//
// N.B. This assumes that the SEQRES records are in order in the PDB file.
func (e *Entry) parseSeqres(line []byte) {
	chain := e.getOrMakeChain(line[11])

	// Residues are in columns 19-21, 23-25, 27-29, ..., 67-69
	for i := 19; i <= 67; i += 4 {
		end := i + 3

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

// parseAtom loads all pertinent information from ATOM records in a PDB file.
// Currently, this only includes deducing the amino acid residue start and
// stop indices. (Note that the length of the range is not necessarily
// equivalent to the length of the amino acid sequence found in the SEQRES
// records.)
//
// ATOM records without a valid amino acid residue in columns 18-20 are ignored.
func (e *Entry) parseAtom(line []byte) {
	chain := e.getOrMakeChain(line[21])

	// An ATOM record is only processed if it corresponds to an amino acid
	// residue. (Which is in columns 17-19.)
	residue := strings.TrimSpace(string(line[17:20]))
	if _, ok := AminoThreeToOne[residue]; !ok {
		// Sanity check. I'm pretty sure that only amino acids have three
		// letter abbreviations.
		if len(residue) == 3 {
			panic(fmt.Sprintf("The residue '%s' found in PDB file '%s' has "+
				"length 3, but is not in my amino acid map.",
				residue, e.Path))
		}
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

// Chain represents a protein chain or subunit in a PDB file. Each chain has
// its own identifier, amino acid sequence (if its a protein sequence), and
// the start and stop residue indices of the ATOM coordinates.
type Chain struct {
	Ident                            byte
	Sequence                         []byte
	AtomResidueStart, AtomResidueEnd int
}

// String returns a FASTA-like formatted string of this chain and all of its
// related information.
func (c *Chain) String() string {
	return fmt.Sprintf("> Chain %c (%d, %d) :: length %d\n%s",
		c.Ident, c.AtomResidueStart, c.AtomResidueEnd,
		len(c.Sequence), string(c.Sequence))
}
