package bowdb

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/pdb"
)

const csvExtras = 3

type files struct {
	db *DB

	bow      *os.File
	bowIndex *os.File
	inverted *os.File
	invIndex inverted
	invCache map[sequenceId]searchItem

	// The following are used to keep state during writing.
	bowIndexOffset int64
	sequenceId     sequenceId
	bufBow         *bytes.Buffer
	csvBow         *csv.Writer
}

func openFiles(db *DB) (fs files, err error) {
	fs.db = db
	fs.invIndex = nil
	fs.invCache = make(map[sequenceId]searchItem, 100)

	if fs.bow, err = db.fileOpen(fileBow); err != nil {
		return
	}
	if fs.bowIndex, err = db.fileOpen(fileBowIndex); err != nil {
		return
	}
	if fs.inverted, err = db.fileOpen(fileInverted); err != nil {
		return
	}
	return
}

func createFiles(db *DB) (fs files, err error) {
	fs.db = db
	fs.bowIndexOffset = 0
	fs.sequenceId = 0
	fs.bufBow = new(bytes.Buffer)
	fs.csvBow = csv.NewWriter(fs.bufBow)
	fs.invIndex = newInvertedIndex(db.Library.Size())

	fs.csvBow.Comma = ','
	fs.csvBow.UseCRLF = false

	if fs.bow, err = db.fileCreate(fileBow); err != nil {
		return
	}
	if fs.bowIndex, err = db.fileCreate(fileBowIndex); err != nil {
		return
	}
	if fs.inverted, err = db.fileCreate(fileInverted); err != nil {
		return
	}
	return
}

func (fs *files) readInvertedSearchItem(fragNum int) ([]searchItem, error) {
	var err error

	// Read in the inverted index if we haven't yet.
	if fs.invIndex == nil {
		if fs.invIndex, err = newInvertedIndexJson(fs.inverted); err != nil {
			return nil, err
		}
	}

	// If the frag num isn't in our index, return an error.
	// Or should we panic? I think it's a bug.
	if fragNum < 0 || fragNum >= len(fs.invIndex) {
		return nil, fmt.Errorf("Fragment number %d is invalid. Valid range "+
			"is [0, %d).", fragNum, len(fs.invIndex))
	}

	// Each fragment corresponds to zero or more pdb chain entries in the
	// BOW database. return them all.
	items := make([]searchItem, len(fs.invIndex[fragNum]))
	for i, seqId := range fs.invIndex[fragNum] {
		if items[i], err = fs.readIndexed(seqId); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (fs *files) newSearchItem(record []string) (searchItem, error) {
	bowMap := make(map[int]int16, fs.db.Library.Size())
	for j := 0; j < fs.db.Library.Size(); j++ {
		n64, err := strconv.ParseInt(record[j+csvExtras], 10, 16)
		if err != nil {
			return searchItem{},
				fmt.Errorf("Could not parse '%d' as a 16-bit integer "+
					"in file '%s' because: %s",
					record[j+csvExtras], fileBow, err)
		}
		bowMap[j] = int16(n64)
	}

	return searchItem{
		PDBItem{
			IdCode:         record[0],
			ChainIdent:     byte(record[1][0]),
			Classification: record[2],
		},
		fs.db.Library.NewBowMap(bowMap),
	}, nil
}

func (fs *files) readIndexed(seqId sequenceId) (searchItem, error) {
	// If we've already read this BOW entry, just return it.
	if si, ok := fs.invCache[seqId]; ok {
		return si, nil
	}

	// Find and seek to the proper place in the BOW database.
	bowOff, err := fs.getBowOffset(seqId)
	if err != nil {
		return searchItem{}, err
	}

	realOff, err := fs.bow.Seek(bowOff, os.SEEK_SET)
	if err != nil {
		return searchItem{},
			fmt.Errorf("Could not seek to %d in the bow database: %s",
				bowOff, err)
	} else if bowOff != realOff {
		return searchItem{},
			fmt.Errorf("Tried to seek to offset %d in the bow database, "+
				"but seeked to %d instead.", bowOff, realOff)
	}

	// Now read the BOW entry.
	csvReader := csv.NewReader(fs.bow)
	csvReader.Comma = ','
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true

	record, err := csvReader.Read()
	if err != nil {
		return searchItem{}, err
	}

	si, err := fs.newSearchItem(record)
	if err != nil {
		return searchItem{}, err
	}

	fs.invCache[seqId] = si
	return si, nil
}

func (fs *files) getBowOffset(seqId sequenceId) (bowOff int64, err error) {
	off := int64(seqId) * 8
	realOff, err := fs.bowIndex.Seek(off, os.SEEK_SET)
	if err != nil {
		return 0, fmt.Errorf("Could not seek to %d in the bow index: %s",
			off, err)
	} else if off != realOff {
		return 0,
			fmt.Errorf("Tried to seek to offset %d in the bow index, "+
				"but seeked to %d instead.", off, realOff)
	}

	err = binary.Read(fs.bowIndex, binary.BigEndian, &bowOff)
	return bowOff, err
}

func (fs *files) read() ([]searchItem, error) {
	reader := csv.NewReader(fs.bow)
	reader.Comma = ','
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	items := make([]searchItem, len(records))
	for i, record := range records {
		if items[i], err = fs.newSearchItem(record); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (fs *files) write(chain *pdb.Chain, bow fragbag.BOW) error {
	endian := binary.BigEndian
	record := make([]string, csvExtras+fs.db.Library.Size())
	record[0] = fmt.Sprintf("%s", chain.Entry.IdCode)
	record[1] = fmt.Sprintf("%c", chain.Ident)
	record[2] = chain.Entry.Classification
	for i := 0; i < fs.db.Library.Size(); i++ {
		record[i+csvExtras] = fmt.Sprintf("%d", bow.Frequency(i))
	}

	fs.bufBow.Reset()
	if err := fs.csvBow.Write(record); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"to the database: %s.", err)
	}
	if err := binary.Write(fs.bowIndex, endian, fs.bowIndexOffset); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"to the database (file '%s'): %s.",
			fs.db.filePath(fileBowIndex), err)
	}

	fs.csvBow.Flush()
	fs.bowIndexOffset += int64(fs.bufBow.Len())
	if _, err := fs.bow.Write(fs.bufBow.Bytes()); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"to the database (file '%s'): %s.", fs.db.filePath(fileBow), err)
	}

	fs.invIndex.add(fs.sequenceId, bow)
	fs.sequenceId++
	return nil
}

func (fs files) writeClose() (err error) {
	if err = fs.bow.Close(); err != nil {
		return
	}
	if err = fs.bowIndex.Close(); err != nil {
		return
	}
	if err = fs.invIndex.write(fs.inverted); err != nil {
		return
	}
	if err = fs.inverted.Close(); err != nil {
		return
	}
	return nil
}

func (fs files) readClose() (err error) {
	if err = fs.bow.Close(); err != nil {
		return
	}
	if err = fs.bowIndex.Close(); err != nil {
		return
	}
	if err = fs.inverted.Close(); err != nil {
		return
	}
	return nil
}
