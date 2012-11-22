/*
Package hhsuite provides convenient wrappers for running programs found in
hhsuite like hhsearch, hhblits and hhmake.

The $HHLIB environment variable is used to determine the location of databases.
i.e., a database named "nr20" will resolve to $HHLIB/data/nr20 and a database
named "fragpred/pdb-select-25" will resolve to
$HHLIB/data/fragpred/pdb-select-25. If this behavior is not desired, change
the global variable DatabasePath to wherever databases are stored. (An empty
database path will leave database named untouched.)

Note that full wrappers for each program are not provided. Options can be added
on an as-needed basis.
*/
package hhsuite
