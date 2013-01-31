package bowdb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

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

func (fs *files) getInvertedList(fragNum int) ([]sequenceId, error) {
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

	return fs.invIndex[fragNum], nil
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
	si, err := fs.readNextBOW()
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
	var item searchItem
	var err error

	if _, err = fs.bow.Seek(0, os.SEEK_SET); err != nil {
		return nil, err
	}

	items := make([]searchItem, 0, 1000)
	for {
		item, err = fs.readNextBOW()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// readNextBOW reads the next entry from the bow file.
// If you're using an index, it is appropriate to call Seek before calling
// readNextBOW.
func (fs *files) readNextBOW() (searchItem, error) {
	// It would be much nicer to use the binary package here (like we do for
	// reading), but we need to be as fast here as possible. (It looks like
	// there is a fair bit of allocation going on in the binary package.)
	libSize := fs.db.Library.Size()
	freqs := make([]int16, libSize)
	entryLenBs := make([]byte, 4)

	// Find the number of bytes that we need to read.
	if n, err := fs.bow.Read(entryLenBs); err != nil {
		if err == io.EOF {
			return searchItem{}, err
		}
		return searchItem{}, fmt.Errorf("Error reading entry length: %s", err)
	} else if n != len(entryLenBs) {
		return searchItem{},
			fmt.Errorf("Expected entry length with length %d, but got %d",
				len(entryLenBs), n)
	}
	entryLen := getUint32(entryLenBs)

	// Read in the full entry.
	entry := make([]byte, entryLen)
	if n, err := fs.bow.Read(entry); err != nil {
		return searchItem{},
			fmt.Errorf("Error reading entry: %s", err)
	} else if n != len(entry) {
		return searchItem{},
			fmt.Errorf("Expected entry with length %d, but got %d",
				len(entry), n)
	}

	// Now gobble up a null terminated id code, a single byte chain identifier,
	// and the BOW vector.
	// Normally we'd use a buffer and the binary package, but we want to be
	// fast.
	idCode := string(entry[0 : len(entry)-(1+1+libSize*2)])
	chainIdent := entry[len(idCode)+1] // account for null terminator
	vector := entry[len(idCode)+1+1:]  // length = libSize * 2
	for i := 0; i < libSize; i++ {
		freqs[i] = getInt16(vector[i*2:])
	}

	return searchItem{
		PDBItem{
			IdCode:     idCode,
			ChainIdent: chainIdent,
		},
		fragbag.BOW{fs.db.Library.Name(), freqs},
	}, nil
}

func (fs *files) write(chain *pdb.Chain, bow fragbag.BOW) error {
	endian := binary.BigEndian
	idCode := fmt.Sprintf("%s%c", chain.Entry.IdCode, 0)
	libSize := fs.db.Library.Size()
	buf := fs.bufBow

	// Write the id code, chain identifier and BOW vector to a buffer.
	fs.bufBow.Reset()
	if _, err := buf.WriteString(idCode); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"id: %s.", err)
	}
	if err := binary.Write(buf, endian, chain.Ident); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"chain id: %s.", err)
	}
	for i := 0; i < libSize; i++ {
		if err := binary.Write(buf, endian, bow.Freqs[i]); err != nil {
			return fmt.Errorf("Something bad has happened when trying to "+
				"write BOW: %s.", err)
		}
	}

	// Write the number of bytes in this entry.
	// (To make reading easier.)
	entryLen := uint32(buf.Len())
	if err := binary.Write(fs.bow, endian, entryLen); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"entry size to (file '%s'): %s.", fs.db.filePath(fileBow), err)
	}

	// Write the buffer to disk.
	if _, err := fs.bow.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"to the database (file '%s'): %s.", fs.db.filePath(fileBow), err)
	}

	// Now update the index.
	if err := binary.Write(fs.bowIndex, endian, fs.bowIndexOffset); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"to the database (file '%s'): %s.",
			fs.db.filePath(fileBowIndex), err)
	}
	fs.bowIndexOffset += int64(entryLen) + 4 // to account for size

	// Add this to the index.
	// (The index isn't written until the end.)
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

// big endian
func getUint32(b []byte) uint32 {
	return uint32(b[0])<<24 |
		uint32(b[1])<<16 |
		uint32(b[2])<<8 |
		uint32(b[3])
}

// big endian
func getInt16(b []byte) int16 {
	return int16(b[0])<<8 | int16(b[1])
}
