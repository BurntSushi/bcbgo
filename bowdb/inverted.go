package bowdb

import (
	"encoding/json"
	"io"

	"github.com/BurntSushi/bcbgo/fragbag"
)

type sequenceId int

type inverted [][]sequenceId

func newInvertedIndex(size int) inverted {
	index := make([][]sequenceId, size)
	for i := 0; i < size; i++ {
		index[i] = make([]sequenceId, 0, 10)
	}
	return index
}

func newInvertedIndexJson(r io.Reader) (inverted, error) {
	index := make([][]sequenceId, 0)
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&index); err != nil {
		return nil, err
	}
	return index, nil
}

func (index inverted) add(sequenceId sequenceId, bow fragbag.BOW) {
	for i := 0; i < bow.Len(); i++ {
		if bow.Freqs[i] == 0 {
			continue
		}
		index[i] = append(index[i], sequenceId)
	}
}

func (index inverted) write(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(index)
}
