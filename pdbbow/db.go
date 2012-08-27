package pdbbow

import (
	"encoding/csv"
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/pdb"
)

const (
	fileConfig   = "config.json"
	fileBow      = "bows"
	fileBowIndex = "bows.index"
	fileInverted = "inverted"
)

type DB struct {
	Library *fragbag.Library
	Config

	path string

	csvBow       *csv.Writer
	fileBow      *os.File
	fileBowIndex *os.File
	fileInverted *os.File
}

func Create(lib *fragbag.Library, path string) (db *DB, err error) {
	// Make sure the DB directory doesn't already exist. If it does, return
	// an error. Otherwise, create the directory.
	_, err = os.Open(path)
	if err == nil {
		return nil,
			fmt.Errorf("Cannot create '%s' directory. It already exists.",
				path)
	}
	if !os.IsNotExist(err) {
		return nil,
			fmt.Errorf("An error occurred when checking if '%s' already "+
				"exists: %s.", path, err)
	}
	if err = os.MkdirAll(path, 0777); err != nil {
		return nil,
			fmt.Errorf("An error occurred when trying to create '%s': %s.",
				path, err)
	}

	db = &DB{
		Library: lib,
		path:    path,
		Config: Config{
			LibraryPath: lib.Path,
		},
	}
	if db.fileBow, err = db.fileCreate(fileBow); err != nil {
		return
	}
	if db.fileBowIndex, err = db.fileCreate(fileBowIndex); err != nil {
		return
	}
	if db.fileInverted, err = db.fileCreate(fileInverted); err != nil {
		return
	}
	db.csvBow = csv.NewWriter(db.fileBow)

	return
}

func (db *DB) Write(entry *pdb.Entry, bow fragbag.BOW) error {
	record := make([]string, 1+db.Library.Size())
	record[0] = entry.Name()
	for i := 0; i < db.Library.Size(); i++ {
		record[i+1] = fmt.Sprintf("%d", bow.Frequency(i))
	}
	if err := db.csvBow.Write(record); err != nil {
		return fmt.Errorf("Something bad has happened when trying to write "+
			"to the database (file '%s'): %s.", db.filePath(fileBow), err)
	}
	return nil
}

func (db *DB) WriteClose() (err error) {
	db.csvBow.Flush()
	if err = db.fileBow.Close(); err != nil {
		return
	}
	if err = db.fileBowIndex.Close(); err != nil {
		return
	}
	if err = db.fileInverted.Close(); err != nil {
		return
	}
	if err = db.Config.write(db.filePath(fileConfig)); err != nil {
		return
	}
	return nil
}

func (db *DB) filePath(name string) string {
	return path.Join(db.path, name)
}

func (db *DB) fileCreate(fname string) (*os.File, error) {
	p := db.filePath(fname)
	f, err := os.Create(p)
	if err != nil {
		return nil, fmt.Errorf("Error creating '%s': %s.", p, err)
	}
	return f, nil
}

func (db *DB) fileOpen(fname string) (*os.File, error) {
	p := db.filePath(fname)
	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("Error opening '%s': %s.", p, err)
	}
	return f, nil
}
