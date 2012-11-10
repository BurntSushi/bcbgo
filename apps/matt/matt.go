package matt

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/BurntSushi/bcbgo/io/pdb"
)

// PDBArg corresponds to an argument to Matt. It can be just a file path
// (represented by the "Path" field), or it can also contain a specific
// chain and a range of residues to align.
type PDBArg struct {
	// Path is the only required field.
	Path string
	IdCode   string
	Chain    byte

	// Matt allows one to specify a range of residues to be aligned.
	// If one of ResidueStart or ResidueEnd is specified, BOTH must be specified
	// in addition to a chain. A violation of this invariant will cause 'Run'
	// to panic.
	ResidueStart, ResidueEnd int
}

// NewPDBArg creates a PDBArg value from a *pdb.Entry. It will fill in the
// Path and IdCode fields. ResidueStart and ResidueEnd will be set to zero.
func NewPDBArg(entry *pdb.Entry) PDBArg {
	return PDBArg{
		Path: entry.Path,
		IdCode: entry.IdCode,
	}
}

// NewChainArg creates a PDBArg value from a *pdb.Chain. It will fill in the
// Path, IdCode and Chain fields. ResidueStart and ResidueEnd will be set
// to zero.
func NewChainArg(chain *pdb.Chain) PDBArg {
	return PDBArg{
		Path: chain.Entry.Path,
		IdCode: chain.Entry.IdCode,
		Chain: chain.Ident,
	}
}

// DefaultConfig provides some sane defaults to run Matt with. For example:
//
//	results, err := matt.DefaultConfig.Run(pdbArg1, pdbArg2, ...)
var DefaultConfig = Config{
	Binary:       "matt",
	OutputPrefix: "",
	Verbose:      true,
	Vomit:        false,
}

// Config is used to specify the location of the Matt binary in addition to
// any relevant parameters that can be passed to Matt. It also controls the
// level of vomit echoed to stderr.
type Config struct {
	// Binary points to the 'matt' executable. If 'matt' is in your PATH,
	// it is sufficient to leave this as 'matt'.
	Binary string

	// OutputPrefix is the parameter given to '-o' in matt. If left blank,
	// the OutputPrefix will be set to a temporary directory in your system's
	// TempDir.
	OutputPrefix string

	// Verbose controls whether all commands executed are printed to stderr.
	Verbose bool

	// When Vomit is true, all vomit from commands executed will also be
	// printed to stderr.
	Vomit bool
}

// RunAll will execute Matt is parallel over all sets of PDB arguments given.
// The order of execution is unspecified, but the order and length of BOTH of
// the return values []*Results and []error is precisely equivalent to the
// order and length of the input [][]PDBArg.
//
// Namely, for all i in [0, (length of [][]PDBArg) - 2], then either
// ([]Results)[i] is not nil EXCLUSIVE OR ([]error)[i] is not nil.
//
// Since this is meant to batch a lot of calls to Matt, several things are
// forcefully automated for you: 1) The OutputPrefix is forced to be empty,
// which results in temporary directories being created for each Matt
// invocation. 2) After each invocation of Matt, 'CleanDir' is used to clean
// up anything leftover by Matt.
//
// If any particular invocation of Matt fails, it is added to the returned
// error slice, but does not stop the overall execution. To determine whether
// invocation 'i' of Matt failed, simply check if the element at index 'i'
// in the returned error slice is 'nil' or not.
//
// Make sure you have GOMAXPROCS set to an appropriate value, or use
// something like:
//
//	runtime.GOMAXPROCS(runtime.NumCPU())
//
// GOMAXPROCS is the maximum number of CPUs that can be executing
// simultaneously. As of July 10, 2012, this value is set by default to 1.
func (conf Config) RunAll(argsets [][]PDBArg) ([]Results, []error) {
	// Force the OutputPrefix to be blank so that we use temp directories.
	conf.OutputPrefix = ""

	// Start N workers, where N is the number of CPUs.
	jobs := make(chan int, 100)
	results := make([]Results, len(argsets))
	errors := make([]error, len(argsets))
	wg := new(sync.WaitGroup)
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for job := range jobs {
				res, err := conf.Run(argsets[job]...)
				res.CleanDir()
				if err != nil {
					errors[job] = err
				} else {
					results[job] = res
				}
			}
		}()
	}
	for i := 0; i < len(argsets); i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	return results, errors
}

// Run will execute Matt using a particular configuration with a set of
// PDB file arguments. It handles creation of a temporary directory where
// Matt writes its output.
//
// After running matt, if you don't intend to use any of the output files,
// you should call 'Clean' (or 'CleanDir' if you want to remove the parent
// directory containing the Matt output files).
//
// After Matt is executed, the core length, RMSD and p-value are read from
// Matt's output 'txt' file automatically. Thus, it is safe to call 'Clean'
// immediately after running, and still have access to those output values
// via the 'Results' struct.
func (conf Config) Run(pargs ...PDBArg) (Results, error) {
	// If the output prefix is blank, we need to set up a temporary directory
	// and use that as the prefix.
	prefix := conf.OutputPrefix
	if len(prefix) == 0 {
		tempDir, err := ioutil.TempDir("", "gomatt")
		if err != nil { // something very bad has happened
			panic(err)
		}
		prefix = fmt.Sprintf("%s/gomatt", tempDir)
	}

	// Now construct the input PDB args.
	args := []string{"-o", prefix}
	for _, parg := range pargs {
		if len(parg.Path) == 0 {
			panic("A PDB argument must have a non-empty location.")
		}
		switch {
		case parg.ResidueStart > 0 || parg.ResidueEnd > 0:
			// Check the invariants. If ResidueStart or ResidueEnd is set, then
			// both must be set and Chain must not be empty.
			if parg.ResidueStart == 0 ||
				parg.ResidueEnd == 0 ||
				parg.Chain == 0 {

				panic("When either ResidueStart or ResidueEnd is set, then " +
					"both must be set, and the Chain must be set.")
			}
			args = append(args, fmt.Sprintf("%s:%c(%d-%d)",
				parg.Path, parg.Chain, parg.ResidueStart, parg.ResidueEnd))
		case parg.Chain != 0:
			args = append(args, fmt.Sprintf("%s:%c", parg.Path, parg.Chain))
		default:
			args = append(args, parg.Path)
		}
	}

	if conf.Verbose {
		fmt.Fprintf(os.Stderr, "%s %s\n", conf.Binary, strings.Join(args, " "))
	}
	out, err := exec.Command(conf.Binary, args...).CombinedOutput()

	if conf.Vomit {
		fmt.Fprintf(os.Stderr, "%s\n", string(out))
	}
	if err != nil {
		return Results{}, fmt.Errorf("%s\n%s", out, err)
	}
	return newResults(prefix)
}

// Results corresponds to information about Matt's output. Use its methods
// "Fasta", "Pdb", "Spt" and "Txt" to retrieve the file names of each of
// Matt's corresponding output files.
type Results struct {
	prefix     string
	CoreLength int
	RMSD       float64
	Pval       float64
}

// newResults uses the Matt prefix to load several interesting pieces of
// information from Matt's txt output file. Namely, the core length, RMSD
// and p-value. newResults returns an error if the output file cannot be read
// or parsed.
func newResults(prefix string) (Results, error) {
	res := Results{prefix: prefix}

	txtf, err := os.Open(res.Txt())
	if err != nil {
		return res, fmt.Errorf("Could not read Matt's txt output file '%s' "+
			"because: %s.", res.Txt(), err)
	}

	txtb, err := ioutil.ReadAll(txtf)
	if err != nil {
		return res, fmt.Errorf("Could not process Matt's txt output file '%s' "+
			"because: %s.", res.Txt(), err)
	}

	txtLines := strings.Split(strings.TrimSpace(string(txtb)), "\n")
	for _, line := range txtLines {
		coloni := strings.Index(line, ":")
		if coloni == -1 {
			continue
		}
		switch line[:coloni] {
		case "Core Residues":
			numStr := strings.TrimSpace(line[coloni+1:])
			if coreLen, err := strconv.ParseInt(numStr, 10, 32); err == nil {
				res.CoreLength = int(coreLen)
			} else {
				return res,
					fmt.Errorf("Could not find core length in line '%s'.", line)
			}
		case "Core RMSD":
			numStr := strings.TrimSpace(line[coloni+1:])
			if rmsd, err := strconv.ParseFloat(numStr, 64); err == nil {
				res.RMSD = rmsd
			} else {
				return res,
					fmt.Errorf("Could not find RMSD in line '%s'.", line)
			}
		case "P-value":
			numStr := strings.TrimSpace(line[coloni+1:])
			if pval, err := strconv.ParseFloat(numStr, 64); err == nil {
				res.Pval = pval
			} else {
				return res,
					fmt.Errorf("Could not find P-value in line '%s'.", line)
			}
		}
	}
	return res, nil
}

// CleanDir will run Clean and also remove the directory containing the Matt
// files. This is usefull when 'Run' is called with an empty OutputPrefix
// (where a temporary directory is created).
func (res Results) CleanDir() {
	res.Clean()
	os.Remove(path.Dir(res.prefix))
}

// Clean will delete all files produced by matt.
// Namely, 'prefix.{fasta,pdb,spt,txt}'. Errors, if they occur, are suppressed.
func (res Results) Clean() {
	os.Remove(res.Fasta())
	os.Remove(res.Pdb())
	os.Remove(res.Spt())
	os.Remove(res.Txt())
}

// Fasta returns the fasta output file path.
func (res Results) Fasta() string {
	return res.prefix + ".fasta"
}

// Pdb returns the PDB output file path.
func (res Results) Pdb() string {
	return res.prefix + ".pdb"
}

// Spt returns the Spt output file path.
func (res Results) Spt() string {
	return res.prefix + ".spt"
}

// Txt returns the txt output file path.
func (res Results) Txt() string {
	return res.prefix + ".txt"
}
