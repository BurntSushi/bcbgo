/*
bestfrag computes the best structural FragBag fragment that corresponds to
the region provided. The region must correspond to at least N alpha-carbon
atoms where N is the size of a fragment in the given fragment library.
The best fragment for each N-sized window in the region provided is echoed
to stdout in this format:

    pdb-id chain-id start end FRAGMENT_NUMBER

where a single space separates each of the 5 fields.

The region specified should be inclusive starting with the number one.

If no region is specified, then the best fragment for every region in the given
chain will be computed.

If no chain is specified, then the best fragment for every chain in the
given PDB file will be computed.

A PDB file may either be plain text or compressed using the Lempel-Ziv coding
(i.e., gzip). If the PDB file is gzipped, it must end with a '.gz' extension.

Usage:
	bestfrag fraglib pdb-file chain-id start stop
*/
package main
