package hhsuite

import (
	"os"
	"path"
	"strings"
)

// The default database path. This will be used to resolve the full paths
// of databases. $HHLIB is usually set in an hhsuite environment.
//
// If you'd like to use a different database path (or none at all), then simply
// change this value to reflect that.
var DatabasePath = path.Join(os.Getenv("HHLIB"), "data")

// A Database is an hhsuite database. A value of type Database should simply
// be the name of the database. i.e., for the $HHBLIB/data/nr20 database, just
// use 'nr20'.
//
// So to use the 'nr20' database, just use 'Database("nr20")'.
//
// If the database ends in '.hhm', then it is assumed to be an hhsearch
// database. Therefore, it cannot work with hhblits (an error will be thrown
// if you try). Otherwise, the database is assumed to be an hhsuite database
// that can be used with hhblits OR hhsearch.
//
// Finally, if the database is an absolute path (i.e., starts with '/'), then
// the database name will be used unaltered.
type Database string

// Resolve will expand a Database value to its full path using DatabasePath.
func (db Database) Resolve() string {
	if path.IsAbs(string(db)) {
		return string(db)
	}
	return path.Join(DatabasePath, string(db))
}

// isOldStyle returns whether this is a database from before hhsuite 2.0.
// i.e., it ends with ".hhm".
func (db Database) isOldStyle() bool {
	return strings.HasSuffix(string(db), ".hhm")
}
