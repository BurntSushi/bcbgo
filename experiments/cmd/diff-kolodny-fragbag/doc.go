/*
diff-kolodny-fragbag is mostly a test program that compares the output of
Kolodny's original implementation of Fragbag with the output of the Fragbag
package. In particular, this output is a bag-of-words fragment vector that
reports the number of times a particular fragment has been the "best match"
(in terms of RMSD) for each K-mer window along the protein backbone (where "K"
in this case is the size of each fragment in the fragment library being used).

In general, this program isn't useful unless you want to compare the two. The
output of this program includes any errors that occur, and a "diff" between
bag-of-words fragment vectors reported by Kolodny's Fragbag and package
fragbag. If there is no difference, "PASSED" is printed.

Usage:
	diff-kolodny-fragbag [flags]
		old-library-file.brk new-library-path pdb-file [ pdb-file ... ]

The flags are:
	--fragbag fragbag-binary-path
		The specified fragbag-binary-path will be used instead of the default
		"fragbag".
	--oldstyle
		When set, NewBowPDBOldStyle will be used to compute BOW vectors for
		package fragbag. See below for more details.

Details

Most of this program is just the grunt work to process the input, pass it
to the fragbag binary and parse the output into a fragbag.BOW value. The
fragbag.BOWDiff type is used to tell whether there is a difference between the
bag-of-words vector returned by Kolodny's Fragbag and package fragbag.

The only important note about this program is that it supports two slightly
different algorithms for computing RMSD in package fragbag. The first mode,
implemented in fragbag.(*Library).NewBowPDB, computes RMSDs for each K-mer
window of *each* chain. Namely, no RMSD is computed for any overlapping K-mer
window between multiple chains. The second mode, implemented in
fragbag.(*Library).NewBowPDBOldStyle, computes RMSDs for each K-mer window of
*all* chains flattened into a single list. Namely, some RMSDs computed will
include K-mer windows that overlap multiple chains.

The second mode exists because it is the algorithm used in Kolodny. It is
implemented in package fragbag so as to provide an apples-to-apples comparison.
The first mode exists because I believe it to be correct. (Results are
forthcoming.)

*/
package main
