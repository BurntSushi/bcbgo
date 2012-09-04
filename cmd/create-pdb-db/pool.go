package main

import (
	"sync"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/pdb"
)

type pool struct {
	wg      *sync.WaitGroup
	entries chan *pdb.Entry
	results chan result
}

type result struct {
	chain *pdb.Chain
	bow   fragbag.BOW
}

func newBowWorkers(lib *fragbag.Library, numWorkers int) pool {
	entries := make(chan *pdb.Entry, numWorkers*2)
	results := make(chan result, numWorkers*2)
	wg := &sync.WaitGroup{}
	for i := 0; i < numWorkers; i++ {
		go func() {
			wg.Add(1)
			for entry := range entries {
				for _, chain := range entry.Chains {
					if chain.ValidProtein() {
						results <- result{chain, lib.NewBowChain(chain)}
					}
				}
			}
			wg.Done()
		}()
	}
	return pool{wg, entries, results}
}

func (p pool) done() {
	close(p.entries)
	p.wg.Wait() // wait for workers to finish sending results
	close(p.results)
}

func (p pool) enqueue(entry *pdb.Entry) {
	p.entries <- entry
}
