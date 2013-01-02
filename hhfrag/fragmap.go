package hhfrag

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/io/fasta"
	"github.com/BurntSushi/bcbgo/io/hhm"
	"github.com/BurntSushi/bcbgo/rmsd"
	"github.com/BurntSushi/bcbgo/seq"
)

type MapConfig struct {
	WindowMin       int
	WindowMax       int
	WindowIncrement int
	Blits           bool
}

var DefaultConfig = MapConfig{
	WindowMin:       6,
	WindowMax:       21,
	WindowIncrement: 3,
	Blits:           false,
}

func getOneFastaSequence(queryFasta string) (s seq.Sequence, err error) {
	fquery, err := os.Open(queryFasta)
	if err != nil {
		return
	}
	defer fquery.Close()

	seqs, err := fasta.NewReader(fquery).ReadAll()
	if err != nil {
		return
	} else if len(seqs) == 0 {
		err = fmt.Errorf("No sequences found in '%s'.", queryFasta)
		return
	} else if len(seqs) > 1 {
		err = fmt.Errorf("%d sequences found in '%s'. Expected only 1.",
			len(seqs), queryFasta)
		return
	}
	s = seqs[0]
	return
}

func (m MapConfig) MapFromFasta(pdbDb PDBDatabase, seqDb hhsuite.Database,
	queryFasta string) (FragmentMap, error) {

	qseq, err := getOneFastaSequence(queryFasta)
	if err != nil {
		return nil, err
	}

	queryHHM, err := hhsuite.BuildHHM(
		hhsuite.HHBlitsDefault, hhsuite.HHMakePseudo, seqDb, queryFasta)
	if err != nil {
		return nil, err
	}
	return m.computeMap(pdbDb, qseq, queryHHM)
}

func (m MapConfig) MapFromHHM(pdbDb PDBDatabase, seqDb hhsuite.Database,
	queryFasta string, queryHHM string) (FragmentMap, error) {

	qseq, err := getOneFastaSequence(queryFasta)
	if err != nil {
		return nil, err
	}

	fquery, err := os.Open(queryHHM)
	if err != nil {
		return nil, err
	}
	defer fquery.Close()

	qhhm, err := hhm.Read(fquery)
	if err != nil {
		return nil, err
	}
	return m.computeMap(pdbDb, qseq, qhhm)
}

func (m MapConfig) computeMap(
	pdbDb PDBDatabase, qseq seq.Sequence, qhhm *hhm.HHM) (FragmentMap, error) {

	type maybeFrag struct {
		frags Fragments
		err   error
	}

	wg := new(sync.WaitGroup)
	jobs := make(chan int, 10)
	fragsChan := make(chan maybeFrag, 10)
	workers := runtime.GOMAXPROCS(0)
	if workers < 1 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		go func() {
			wg.Add(1)
			defer wg.Done()

			min, max := m.WindowMin, m.WindowMax
		CHANNEL:
			for start := range jobs {
				var best *Fragments
				for end := min; end <= max && (start+end) <= qseq.Len(); end++ {
					frags, err := FindFragments(
						pdbDb, m.Blits, qhhm, qseq, start, start+end)
					if err != nil {
						fragsChan <- maybeFrag{
							err: err,
						}
						continue CHANNEL
					}
					if best == nil || frags.better(*best) {
						best = frags
					}
				}
				fragsChan <- maybeFrag{
					frags: *best,
				}
			}
		}()
	}
	go func() {
		for s := 0; s <= qseq.Len()-m.WindowMin; s += m.WindowIncrement {
			jobs <- s
		}
		close(jobs)
		wg.Wait()
		close(fragsChan)
	}()

	fmap := make(FragmentMap, 0, 50)
	for maybeFrag := range fragsChan {
		if maybeFrag.err != nil {
			return nil, maybeFrag.err
		}
		fmap = append(fmap, maybeFrag.frags)
	}
	sort.Sort(fmap)
	return fmap, nil
}

type FragmentMap []Fragments

func (fmap FragmentMap) Len() int {
	return len(fmap)
}

func (fmap FragmentMap) Less(i, j int) bool {
	return fmap[i].Start < fmap[j].Start
}

func (fmap FragmentMap) Swap(i, j int) {
	fmap[i], fmap[j] = fmap[j], fmap[i]
}

func (fmap FragmentMap) BOW(lib *fragbag.Library) fragbag.BOW {
	bow := lib.NewBow()
	mem := rmsd.NewQcMemory(lib.FragmentSize())
	for _, fragGroup := range fmap {
		for _, frag := range fragGroup.Frags {
			if len(frag.CaAtoms) < lib.FragmentSize() {
				continue
			}
			for i := 0; i <= len(frag.CaAtoms)-lib.FragmentSize(); i++ {
				atoms := frag.CaAtoms[i : i+lib.FragmentSize()]
				bestFragNum, _ := lib.BestFragment(atoms, mem)
				bow.Increment(bestFragNum)
			}
		}
	}
	return bow
}
