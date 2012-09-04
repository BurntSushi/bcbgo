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

const csvExtras = 2

type files struct {
	db *DB

	bow      *os.File
	bowIndex *os.File
	inverted *os.File
	invIndex inverted

	// The following are used to keep state during writing.
	bowIndexOffset int64
	sequenceId     sequenceId
	bufBow         *bytes.Buffer
	csvBow         *csv.Writer
}

func openFiles(db *DB) (fs files, err error) {
	fs.db = db
	fs.invIndex = newInvertedIndex(db.Library.Size())

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
		bowMap := make(map[int]int16, fs.db.Library.Size())
		for j := 0; j < fs.db.Library.Size(); j++ {
			n64, err := strconv.ParseInt(record[j+csvExtras], 10, 16)
			if err != nil {
				return nil,
					fmt.Errorf("Could not parse '%d' as a 16-bit integer "+
						"in file '%s' because: %s",
						record[j+csvExtras], fileBow, err)
			}
			bowMap[j] = int16(n64)

			items[i] = searchItem{
				PDBItem{
					IdCode:         record[0],
					Classification: record[1],
				},
				fs.db.Library.NewBowMap(bowMap),
			}
		}
	}
	return items, nil
}

func (fs *files) write(chain *pdb.Chain, bow fragbag.BOW) error {
	endian := binary.BigEndian
	record := make([]string, csvExtras+fs.db.Library.Size())
	record[0] = fmt.Sprintf("%s%c", chain.Entry.IdCode, chain.Ident)
	record[1] = chain.Entry.Classification
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
