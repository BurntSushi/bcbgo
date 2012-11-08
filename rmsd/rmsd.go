package rmsd

import (
	"fmt"
	"math"

	"github.com/BurntSushi/bcbgo/pdb"

	matrix "github.com/skelterjohn/go.matrix"
)

// RMSD implements a version of the Kabsch alogrithm that is described here:
// http://cnx.org/content/m11608/latest/
//
// A brief, high-level overview:
//
// Build the 3xN matrices X and Y containing, for the sets x and y 
// respectively, the coordinates for each of the N atoms after centering 
// the atoms by subtracting the centroids. 
//  
// Compute the covariance matrix C=X(Y^T) 
//  
// Compute the SVD (Singular Value Decomposition) of C=VS(W^T) 
//  
// Compute d=sign(det(C)) 
//  
// Compute the optimal rotation U as U = W([1 0 0] [0 1 0] [0 0 d])(V^T) 
//
// (In the last step, we're using WT instead of W, since that's apparently
// what Fragbag does, and we're looking to imitate them. At least, at first!)
//
// Note that RMSD will panic if the lengths of struct1 and struct2 differ.
// RMSD will also panic if the calculation of the SVD returns an error. (It's
// possible that will change, though.)
func RMSD(struct1, struct2 pdb.Atoms) float64 {
	if len(struct1) != len(struct2) {
		panic(fmt.Sprintf("Computing the RMSD of two structures require that "+
			"they have equal length. But the lengths of the two structures "+
			"provided are %d and %d.", len(struct1), len(struct2)))
	}

	// In order to "center" the coordinates, we
	// subtract the centroid for each set of atom coordinates.
	cx1, cy1, cz1 := centroid(struct1)
	cx2, cy2, cz2 := centroid(struct2)

	// Initialize the go.matrix values from the 3 dimensional ATOM coordinates.
	// We end up with two 3xN matrices (X and Y), where N is the length of
	// struct1 and struct2.
	cols := len(struct1)
	X := make([]float64, 3*cols)
	Y := make([]float64, 3*cols)
	for i := 0; i < len(struct1); i++ {
		a1, a2 := struct1[i].Coords, struct2[i].Coords
		X[0*cols+i] = a1[0] - cx1
		X[1*cols+i] = a1[1] - cy1
		X[2*cols+i] = a1[2] - cz1

		Y[0*cols+i] = a2[0] - cx2
		Y[1*cols+i] = a2[1] - cy2
		Y[2*cols+i] = a2[2] - cz2
	}

	// Compute the covariance matrix C = X(Y^T)
	C := covariant_3x3(cols, X, Y)

	// Compute the Singular Value Decomposition of C = VS(W^T)
	V, WT := C.svd()

	// If the determinant of C is negative, then we have to correct for
	// something called an "improper rotation" in that the matrix doesn't
	// constitute a "right handed system". To correct for it, we multiply
	// W by ( [1 0 0] [0 1 0] [0 0 -1] ). This makes the rotation "proper".
	// When the determinant is positive, we save some cycles and just
	// compute W(V^T)
	//
	// N.B. We are using WT here, even though the algorithm described at
	// http://cnx.org/content/m11608/latest/
	// calls for W. For whatever reason, Fragbag's algorithm seems to use
	// WT. (i.e., this approach matches its output precisely.)
	VT := V.transpose()
	if C.det() < 0 {
		adjust := matrix3{
			1, 0, 0,
			0, 1, 0,
			0, 0, -1,
		}
		WT = WT.mult(adjust)
	}
	U := WT.mult(VT)

	// Apply the rotational matrix U to X to get the best possible alignment
	// with Y.
	Xbest := mult_3x3_3xN(cols, U[:], X)

	// Now compute the RMSD between Xbest and Y.
	var rmsd, dist float64 = 0.0, 0.0
	for r := 0; r < 3; r++ {
		for c := 0; c < cols; c++ {
			dist = Xbest[r*cols+c] - Y[r*cols+c]
			rmsd += dist * dist
		}
	}
	return math.Sqrt(rmsd / float64(len(struct1)))
}

// centroid calculates the average position of a set of atoms.
func centroid(atoms pdb.Atoms) (float64, float64, float64) {
	var xs, ys, zs float64
	for _, atom := range atoms {
		xs += atom.Coords[0]
		ys += atom.Coords[1]
		zs += atom.Coords[2]
	}
	n := float64(len(atoms))
	return xs / n, ys / n, zs / n
}

// must panics if the result of a dense matrix operation returns an error.
func must(A *matrix.DenseMatrix, err error) *matrix.DenseMatrix {
	if err != nil {
		panic(err)
	}
	return A
}
