package fragbag

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/bcbgo/io/pdb"
)

// Library represents a Fragbag fragment library. Fragbag fragment libraries
// are fixed both in the number of fragments and in the size of each fragment.
//
// A Fragbag library traditionally existed as one monolithic '.brk' file, but
// this implementation uses a slightly different form called General Fragment
// Form. The key distinction is that a library is represented on disk as a
// directory where each fragment in the library corresponds to a single file
// with a 16-bit integer name. (i.e., "4" and "2000" are valid fragment file
// names, but "one" and "1000000" are not.)
//
// The only invariants imposed by this package is that every fragment in a
// library must be the same size, and fragment numbers must start from 1 and
// be contiguous.
//
// Files in a Fragbag library directory that do not correspond to 16-bit
// integer names are simply ignored.
type Library struct {
	Path         string
	fragmentSize int
	fragments    []Fragment
}

// NewLibrary constucts a new Fragbag library given a path to the fragment
// library file.
// It will return an error if the file is not readable, if there are no
// fragments, or if there is any variability among fragment sizes in all
// fragments.
func NewLibrary(fpath string) (*Library, error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	lib := &Library{
		Path:         fpath,
		fragmentSize: 0,
		fragments:    make([]Fragment, 0),
	}

	fragments := bytes.Split(contents, []byte("TER"))
	flen := 0
	for i, fragment := range fragments {
		fragment = bytes.TrimSpace(fragment)
		if len(fragment) == 0 {
			continue
		}
		frag, err := lib.newFragment(i, fragment)
		if err != nil {
			return nil, fmt.Errorf("Could not read fragment '%d': %s", i, err)
		}
		if len(frag.CaAtoms) == 0 {
			return nil, fmt.Errorf("No Ca atoms for fragment '%d'.", i)
		}
		if flen == 0 {
			flen = len(frag.CaAtoms)
		} else if flen != len(frag.CaAtoms) {
			return nil,
				fmt.Errorf("Fragment '%d' has length %d, but others have "+
					"length %d.", i, len(frag.CaAtoms), flen)
		}
		lib.fragments = append(lib.fragments, frag)
	}
	lib.fragmentSize = flen
	return lib, nil
}

// Copy copies the full fragment library to the path provied.
func (lib *Library) Copy(dest string) error {
	fdest, err := os.Create(dest)
	if err != nil {
		return err
	}

	fsrc, err := os.Open(lib.Path)
	if err != nil {
		return err
	}

	if _, err := io.Copy(fdest, fsrc); err != nil {
		return err
	}

	return nil
}

// Size returns the number of fragments in the library.
func (lib *Library) Size() int {
	return len(lib.fragments)
}

// FragmentSize returns the size of each fragment.
func (lib *Library) FragmentSize() int {
	return lib.fragmentSize
}

func (lib *Library) Fragments() []Fragment {
	return lib.fragments
}

// Fragment returns a LibFragment corresponding to the fragment number
// fragNum. Fragment will panic if such a fragment does not exist.
func (lib *Library) Fragment(fragNum int) Fragment {
	if fragNum >= 0 && fragNum < len(lib.fragments) {
		return lib.fragments[fragNum]
	}
	panic(fmt.Sprintf("Fragment number %d does not exist in the "+
		"'%s' fragment library.", fragNum, lib))
}

// String returns a string with the name of the library (base name of the
// library directory), the number of fragments in the library and the size
// of each fragment.
func (lib *Library) String() string {
	return fmt.Sprintf("%s (%d, %d)",
		path.Base(lib.Path), len(lib.fragments), lib.fragmentSize)
}

func (lib *Library) Name() string {
	return path.Base(lib.Path)
}

// Fragment corresponds to a single fragment file in a fragment library.
// It holds the fragment number identifier and embeds a PDB entry.
type Fragment struct {
	library *Library
	Ident   int
	*pdb.Entry
	CaAtoms []pdb.Coords
}

// newLibFragment creates a new LibFragment with a particular fragment number
// given a fragment library. A pdb.Entry is also created and embedded with
// the Location corresponding to a file path concatenation of the library path
// and the fragment number.
func (lib *Library) newFragment(fragNum int, frag []byte) (Fragment, error) {
	entry, err := pdb.Read(
		bytes.NewReader(frag), fmt.Sprintf("%s:%d", lib.Path, fragNum))
	if err != nil {
		return Fragment{}, err
	}
	return Fragment{
		library: lib,
		Ident:   fragNum,
		Entry:   entry,
		CaAtoms: entry.OneChain().CaAtoms(),
	}, nil
}

// String returns the fragment number, library and its corresponding atoms.
func (frag *Fragment) String() string {
	atoms := make([]string, len(frag.CaAtoms))
	for i, atom := range frag.CaAtoms {
		atoms[i] = fmt.Sprintf("\t%s", atom)
	}
	return fmt.Sprintf("> %d (%s)\n%s",
		frag.Ident, frag.library, strings.Join(atoms, "\n"))
}
