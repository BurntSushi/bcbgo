package hhfrag

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/io/fasta"
	"github.com/BurntSushi/bcbgo/io/hhm"
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
	fragsChan := make(chan maybeFrag, 1)
	fmap := make(FragmentMap, 0, 25)
	for i := 0; i <= qseq.Len()-m.WindowMin; i += 3 {
		i := i
		go func() {
			wg.Add(1)
			defer wg.Done()

			var best *Fragments
			for j := m.WindowMin; j <= m.WindowMax && (i+j) <= qseq.Len(); j++ {
				frags, err := FindFragments(pdbDb, m.Blits, qhhm, qseq, i, i+j)
				if err != nil {
					fragsChan <- maybeFrag{
						err: err,
					}
					return
				}
				if best == nil || frags.better(*best) {
					best = frags
				}
			}
			fragsChan <- maybeFrag{
				frags: *best,
			}
		}()
	}

	go func() {
		wg.Wait()
		close(fragsChan)
	}()
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
