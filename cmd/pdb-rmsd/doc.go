/*
pdb-rmsd computes the RMSD between two sets of carbon-alpha ATOM records read 
from PDB files. Namely, each set of ATOM records is specified by a four-tuple: 
a PDB file path, a chain identifier, and an inclusive range of residue indices. 
Notably, both sets of cabon-alpha ATOM records must be exactly the same size.

A PDB file may either be plain text or compressed using the Lempel-Ziv coding 
(i.e., gzip). If the PDB file is gzipped, it must end with a '.gz' extension.

Usage:
	pdb-rmsd pdb-file chain-id start stop pdb-file chain-id start stop

Details

The algorithm used to compute RMSD is based on the Kabsch algorithm for 
computing the optimal rotational matrix which minimizes RMSD between two paired 
sets of points.

More precisely, the algorithm is described in great detail here:
http://cnx.org/content/m11608/latest/#MatrixAlignment.
*/
package documentation
