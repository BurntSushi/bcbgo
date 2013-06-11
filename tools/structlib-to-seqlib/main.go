package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
	"strings"
	"sync"

	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/io/pdb"
	"github.com/BurntSushi/bcbgo/io/pdb/slct"
	"github.com/BurntSushi/bcbgo/seq"
)

var (
	flagOverwrite = false
	flagPdbSelect = false
)

var (
	// There are two concurrent aspects going on here:
	// 1) processing entire PDB chains
	// 2) adding each part of each chain to a sequence fragment.
	// So we use two waitgroups: one for synchronizing on finishing
	// (1) and the other for synchronizing on finishing (2).
	wgPDBChains    = new(sync.WaitGroup)
	wgSeqFragments = new(sync.WaitGroup)

	// The structure library supplied by the user.
	structLib *fragbag.StructureLibrary
)

func init() {
	flag.BoolVar(&flagOverwrite, "overwrite", flagOverwrite,
		"When set, any existing database will be completely overwritten.")
	flag.BoolVar(&flagPdbSelect, "pdb-select", flagPdbSelect,
		"When set, the protein list will be read as a PDB Select file.")

	util.FlagUse("cpu", "cpuprof", "verbose")
	util.FlagParse(
		"struct-lib protein-list seq-lib-outfile",
		"Where 'protein-list' is a plain text file with PDB chain\n"+
			"identifiers on each line. e.g., '1P9GA'.")
	util.AssertLeastNArg(3)
}

func main() {
	if len(util.FlagCpuProf) > 0 {
		f := util.CreateFile(util.FlagCpuProf)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	libPath := util.Arg(0)
	protList := util.Arg(1)
	saveto := util.Arg(2)

	structLib = util.StructureLibrary(libPath)
	util.AssertOverwritable(saveto, flagOverwrite)

	// Initialize a frequency profile for each structural fragment.
	var freqProfiles []*seq.FrequencyProfile
	var fpChans []chan seq.Sequence
	for i := 0; i < structLib.Size(); i++ {
		fp := seq.NewFrequencyProfile(structLib.FragmentSize)
		freqProfiles = append(freqProfiles, fp)

		fpChan := make(chan seq.Sequence)
		fpChans = append(fpChans, fpChan)
	}

	// Also initialize a special frequency profile for the null model.
	nullProfile := seq.NewNullProfile()
	nullChan := make(chan seq.Sequence)
	addToNullProfile(nullChan, nullProfile)

	// Now spin up a goroutine for each fragment that is responsible for
	// adding a sequence slice to itself.
	for i := 0; i < structLib.Size(); i++ {
		addToProfile(fpChans[i], freqProfiles[i])
	}

	chainIds, numChainIds := genChains(protList)
	progressChan := progress(numChainIds)
	for i := 0; i < util.FlagCpu; i++ {
		wgPDBChains.Add(1)
		go func() {
			for chainId := range chainIds {
				progressChan <- struct{}{}

				pdbPath := util.PDBPath(chainId)
				if !util.Exists(pdbPath) {
					util.Verbosef("PDB file '%s' from chain '%s' does "+
						"not exist.", pdbPath, chainId)
					continue
				}
				_, chain := util.PDBReadId(chainId)
				structureToSequence(chain, nullChan, fpChans)
			}
			wgPDBChains.Done()
		}()
	}
	wgPDBChains.Wait()

	// We've finishing reading all the PDB inputs. Now close the channels
	// and let the sequence fragments finish.
	for i := 0; i < structLib.Size(); i++ {
		close(fpChans[i])
	}
	close(nullChan)
	wgSeqFragments.Wait()

	// Finally, add the sequence fragments to a new sequence fragment
	// library and save.
	seqLib := fragbag.NewSequenceLibrary(structLib.Ident)
	for i := 0; i < structLib.Size(); i++ {
		p := freqProfiles[i].Profile(nullProfile)
		util.Assert(seqLib.Add(p))
	}
	util.Assert(seqLib.Save(util.CreateFile(saveto)))
}

// structureToSequence uses structural fragments to categorize a segment
// of alpha-carbon atoms, and adds the corresponding residues to a
// corresponding sequence fragment.
func structureToSequence(
	chain *pdb.Chain,
	nullChan chan seq.Sequence,
	fpChans []chan seq.Sequence,
) {
	sequence := chain.AsSequence()

	// If the chain is shorter than the fragment size, we can do nothing
	// with it.
	if sequence.Len() < structLib.FragmentSize {
		util.Verbosef("Sequence '%s' is too short (length: %d)",
			sequence.Name, sequence.Len())
		return
	}

	limit := sequence.Len() - structLib.FragmentSize
	for start := 0; start <= limit; start++ {
		end := start + structLib.FragmentSize
		atoms := chain.SequenceCaAtomSlice(start, end)
		if atoms == nil {
			// Nothing contiguous was found (a "disordered" residue perhaps).
			// So skip this part of the chain.
			continue
		}
		bestFrag := structLib.Best(atoms)

		sliced := sequence.Slice(start, end)
		fpChans[bestFrag] <- sliced
		nullChan <- sliced
	}
}

func addToProfile(sequences chan seq.Sequence, fp *seq.FrequencyProfile) {
	wgSeqFragments.Add(1)
	go func() {
		for s := range sequences {
			fp.Add(s)
		}
		wgSeqFragments.Done()
	}()
}

func addToNullProfile(sequences chan seq.Sequence, fp *seq.FrequencyProfile) {
	wgSeqFragments.Add(1)
	go func() {
		for s := range sequences {
			for i := 0; i < s.Len(); i++ {
				fp.Add(s.Slice(i, i+1))
			}
		}
		wgSeqFragments.Done()
	}()
}

func genChains(protList string) (chan string, int) {
	ids := make([]string, 0, 100)
	file := util.OpenFile(protList)
	if flagPdbSelect {
		records, err := slct.NewReader(file).ReadAll()
		util.Assert(err)
		for _, r := range records {
			if len(r.ChainID) != 5 {
				util.Fatalf("Not a valid chain identifier: '%s'", r.ChainID)
			}
			ids = append(ids, r.ChainID)
		}
	} else {
		for _, line := range util.ReadLines(file) {
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			} else if len(line) != 5 {
				util.Fatalf("Not a valid chain identifier: '%s'\n"+
					"Perhaps you forgot to set 'pdb-select'?", line)
			}
			ids = append(ids, line)
		}
	}

	// Convert chain IDs to a channel.
	// Idea: multiple goroutines can read and parse PDB files in parallel.
	chains := make(chan string)
	go func() {
		for _, id := range ids {
			chains <- id
		}
		close(chains)
	}()
	return chains, len(ids)
}

func progress(total int) chan struct{} {
	count := 0
	c := make(chan struct{})
	pf := func(ft string, v ...interface{}) { fmt.Fprintf(os.Stderr, ft, v...) }
	go func() {
		for _ = range c {
			count++
			pf("\r%d/%d (%0.2f%% complete)", count, total,
				100.0*(float64(count)/float64(total)))
		}
	}()
	return c
}
