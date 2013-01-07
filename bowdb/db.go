package bowdb

import (
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

const (
	fileConfig   = "config.json"
	fileBow      = "bows"
	fileBowIndex = "bows.index"
	fileInverted = "inverted"
)

const (
	SearchFull = iota
	SearchInverted
)

type DB struct {
	Library *fragbag.Library
	Config

	path  string
	files files
}

func Open(path string) (db *DB, err error) {
	if _, err = os.Open(path); err != nil {
		return nil, fmt.Errorf("Cannot open '%s' directory: %s", path, err)
	}

	db = &DB{
		path: path,
	}
	db.Config, err = openConfig(db.filePath(fileConfig))
	if err != nil {
		return
	}

	db.Library, err = fragbag.NewLibrary(db.LibraryPath)
	if err != nil {
		return
	}

	db.files, err = openFiles(db)
	if err != nil {
		return
	}
	return
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
		Config: Config{
			LibraryPath: lib.Path,
		},
		path: path,
	}

	db.files, err = createFiles(db)
	if err != nil {
		return
	}
	return
}

func (db *DB) Write(chain *pdb.Chain, bow fragbag.BOW) error {
	return db.files.write(chain, bow)
}

func (db *DB) WriteClose() (err error) {
	if err = db.Config.write(db.filePath(fileConfig)); err != nil {
		return
	}
	return db.files.writeClose()
}

type Entry struct {
	IdCode     string
	ChainIdent byte
	BOW        fragbag.BOW
}

func (db *DB) ReadAll() ([]Entry, error) {
	searchItems, err := db.files.read()
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(searchItems))
	for i, si := range searchItems {
		entries[i] = Entry{
			IdCode:     si.IdCode,
			ChainIdent: si.ChainIdent,
			BOW:        si.BOW,
		}
	}
	return entries, nil
}

func (db *DB) ReadClose() (err error) {
	return db.files.readClose()
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
