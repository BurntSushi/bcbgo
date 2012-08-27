package bowdb

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/pdb"
)

type files struct {
	db *DB

	bow            *os.File
	bowIndex       *os.File
	inverted       *os.File
	bowIndexOffset int64
	sequenceId     sequenceId
	bufBow         *bytes.Buffer
	csvBow         *csv.Writer
	invIndex       inverted
}

func createFiles(db *DB) (fs files, err error) {
	fs.db = db
	fs.bowIndexOffset = 0
	fs.sequenceId = 0
	fs.bufBow = new(bytes.Buffer)
	fs.csvBow = csv.NewWriter(fs.bufBow)
	fs.invIndex = newInvertedIndex(db.Library.Size())

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

func (fs *files) write(entry *pdb.Entry, bow fragbag.BOW) error {
	endian := binary.BigEndian
	extras := 2
	record := make([]string, extras+fs.db.Library.Size())
	record[0] = entry.IdCode
	record[1] = entry.Classification
	for i := 0; i < fs.db.Library.Size(); i++ {
		record[i+extras] = fmt.Sprintf("%d", bow.Frequency(i))
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
