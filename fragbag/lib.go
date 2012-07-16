package fragbag

import (
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/bcbgo/pdb"
	"github.com/BurntSushi/bcbgo/rmsd"
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
	fragments    []*LibFragment
}

// NewLibrary constucts a new Fragbag library given a path to the directory
// containing the fragment files. It will return an error if the directory
// or any of the fragment files aren't readable, if there are no fragment files,
// or if there is any variability among fragment sizes in all fragments.
func NewLibrary(path string) (*Library, error) {
	infos, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	lib := &Library{
		Path:         path,
		fragmentSize: 0,
		fragments:    nil,
	}

	// Add all files in the directory that have names that are translatable
	// to 16-bit integers.
	fragNums := make([]int, 0, 100)
	for _, info := range infos {
		if fragNum64, err := strconv.ParseInt(info.Name(), 10, 16); err == nil {
			fragNums = append(fragNums, int(fragNum64))
		}
	}

	// If there aren't any fragments, return an error. No empty libraries!
	if len(fragNums) == 0 {
		return nil, fmt.Errorf("The library at '%s' does not contain any "+
			"fragments.", path)
	}

	// Check that the fragment numbering is contiguous, and populate
	// our fragments slice.
	lib.fragments = make([]*LibFragment, len(fragNums))
	sort.Sort(sort.IntSlice(fragNums))
	lastNum := -1
	for _, fragNum := range fragNums {
		if lastNum != fragNum-1 {
			if lastNum == -1 {
				return nil, fmt.Errorf("Fragment files must start numbering "+
					"at 1 and be contiguous. However, the first fragment "+
					"number in library '%s' is %d.",
					path, fragNum)
			}
			return nil, fmt.Errorf("Fragment files must start numbering "+
				"at 1 and be contiguous. However, a fragment of number "+
				"%d was found after a fragment of number %d.",
				fragNum, lastNum)
		}
		lastNum = fragNum

		lib.fragments[fragNum], err = lib.newLibFragment(fragNum)
		if err != nil {
			return nil, fmt.Errorf("The fragment file '%d' in the '%s' "+
				"library could not be read as a PDB file: %s",
				fragNum, path, err)
		}
	}

	// Set the fragment size of this library to the size of one of the
	// fragments. Then make sure every other fragment has the same size.
	for _, frag := range lib.fragments {
		size := len(frag.OneChain().CaAtoms)
		if lib.fragmentSize == 0 {
			lib.fragmentSize = size
		} else {
			if size != lib.fragmentSize {
				return nil, fmt.Errorf("In the library at '%s', fragment %d "+
					"has a size of %d, but another fragment has a size of %d.",
					path, frag.Ident, size, lib.fragmentSize)
			}
		}
	}

	return lib, nil
}

// Size returns the number of fragments in the library.
func (lib *Library) Size() int {
	return len(lib.fragments)
}

// FragmentSize returns the size of each fragment.
func (lib *Library) FragmentSize() int {
	return lib.fragmentSize
}

// Fragment returns a LibFragment corresponding to the fragment number
// fragNum. Fragment will panic if such a fragment does not exist.
func (lib *Library) Fragment(fragNum int) *LibFragment {
	if fragNum >= 0 && fragNum < len(lib.fragments) {
		return lib.fragments[fragNum]
	}
	panic(fmt.Sprintf("Fragment number %d does not exist in the "+
		"'%s' fragment library.", fragNum, lib))
}

// BestFragment runs Kabsch using the provided PDB argument against all 
// fragments in the library and returns the fragment number with the best RMSD.
//
// BestFragment panics if the length of atoms is not equivalent to the
// fragment size of the library.
func (lib *Library) BestFragment(atoms pdb.Atoms) int {
	if len(atoms) != lib.FragmentSize() {
		panic(fmt.Sprintf("BestFragment can only be called with a list of "+
			"atoms with length equivalent to the fragment size of the "+
			"library. The length of the list given is %d, but the fragment "+
			"size of the library is %d.", len(atoms), lib.FragmentSize()))
	}

	bestRmsd, bestFragNum := 0.0, -1
	for _, frag := range lib.fragments {
		testRmsd := rmsd.RMSD(atoms, frag.OneChain().CaAtoms)
		if bestFragNum == -1 || testRmsd < bestRmsd {
			bestRmsd, bestFragNum = testRmsd, frag.Ident
		}
	}
	return bestFragNum
}

type rmsdPoolJob struct {
	fragNum int
	atoms   pdb.Atoms
}

type rmsdPoolResult struct {
	fragNum int
	rmsd    float64
}

// BestFragments runs Kabsch in parallel over the entire fragment library
// for *each* set of atoms provided. The best fragment for each atom set
// is returned in a slice of integers whose indices correspond exactly to
// the indices of 'atomSets'.
func (lib *Library) BestFragments(atomSets []pdb.Atoms) []int {
	// Create the worker pool.
	jobs, results := lib.rmsdWorkers(0, 0)
	bestFrags := make([]int, len(atomSets))

	for setIndex, atoms := range atomSets {
		if len(atoms) != lib.FragmentSize() {
			panic(fmt.Sprintf("BestFragments can only be called with sets "+
				"of atoms with length equivalent to the fragment size of the "+
				"library. The length a set given is %d, but the "+
				"fragment size of the library is %d.",
				len(atoms), lib.FragmentSize()))
		}
		// Start a goroutine that reads the results returned from the worker
		// pool, and determines the best matching fragment in terms of
		// smallest RMSD.
		bestFragChan := make(chan int, 0)
		go func() {
			bestFrag, bestRmsd := -1, 0.0
			for i := 0; i < lib.Size(); i++ {
				result := <-results
				if bestFrag == -1 || result.rmsd < bestRmsd {
					bestFrag, bestRmsd = result.fragNum, result.rmsd
				}
			}
			bestFragChan <- bestFrag
		}()

		// Send out all of the jobs to the workers.
		for i := 0; i < lib.Size(); i++ {
			jobs <- rmsdPoolJob{
				fragNum: i,
				atoms:   atoms,
			}
		}

		// Wait for the best fragment computation to finish, then move on to
		// the next atom set.
		//
		// This is CRITICAL! If we don't wait (i.e., slapping this into
		// a goroutine), then it's quite likely that results from one atom set
		// will get confused for another atom set.
		bestFrags[setIndex] = <-bestFragChan
	}
	return bestFrags
}

// rmsdWorkers starts a pool of workers ready to compute the RMSD of any two
// atom slices. If 'numWorkers' is 0, then GOMAXPROCS workers will be started.
// If 'bufferSize' is 0, then the buffer will be set to the number of fragments
// in the library. The bigger the buffer, the more memory will be used.
func (lib *Library) rmsdWorkers(
	numWorkers int, bufferSize int) (chan rmsdPoolJob, chan rmsdPoolResult) {

	workers := numWorkers
	if workers == 0 {
		workers = runtime.GOMAXPROCS(0)
	}
	bufSize := bufferSize
	if bufSize == 0 {
		bufSize = lib.Size()
	}

	jobs := make(chan rmsdPoolJob, bufSize)
	results := make(chan rmsdPoolResult, bufSize)
	for i := 0; i < workers; i++ {
		go func() {
			for job := range jobs {
				results <- rmsdPoolResult{
					fragNum: job.fragNum,
					rmsd: rmsd.RMSD(
						job.atoms,
						lib.Fragment(job.fragNum).OneChain().CaAtoms),
				}
			}
		}()
	}
	return jobs, results
}

// String returns a string with the name of the library (base name of the 
// library directory), the number of fragments in the library and the size
// of each fragment.
func (lib *Library) String() string {
	return fmt.Sprintf("%s (%d, %d)",
		path.Base(lib.Path), len(lib.fragments), lib.fragmentSize)
}

// mustExist will panic if the given fragment number does not exist.
func (lib *Library) mustExist(fragNum int) {
	if fragNum < 0 || fragNum >= len(lib.fragments) {
		panic(fmt.Sprintf("Fragment number %d does not exist in library '%s'.",
			fragNum, lib))
	}
}

// LibFragment corresponds to a single fragment file in a fragment library.
// It holds the fragment number identifier and embeds a PDB entry.
type LibFragment struct {
	library *Library
	Ident   int
	*pdb.Entry
}

// newLibFragment creates a new LibFragment with a particular fragment number
// given a fragment library. A pdb.Entry is also created and embedded with
// the Location corresponding to a file path concatenation of the library path
// and the fragment number.
func (lib *Library) newLibFragment(fragNum int) (*LibFragment, error) {
	path := path.Join(lib.Path, fmt.Sprintf("%d", fragNum))
	entry, err := pdb.New(path)
	if err != nil {
		return nil, err
	}
	return &LibFragment{
		library: lib,
		Ident:   fragNum,
		Entry:   entry,
	}, nil
}

// String returns the fragment number, library and its corresponding atoms.
func (frag *LibFragment) String() string {
	chain := frag.OneChain()
	atoms := make([]string, len(chain.CaAtoms))
	for i, atom := range chain.CaAtoms {
		atoms[i] = fmt.Sprintf("\t%s", atom)
	}
	return fmt.Sprintf("> %d (%s)\n%s",
		frag.Ident, frag.library, strings.Join(atoms, "\n"))
}
