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
	els1 := make([]float64, 3*len(struct1))
	els2 := make([]float64, 3*len(struct2))
	for i := 0; i < len(struct1); i++ {
		o := len(struct1)
		a1, a2 := struct1[i].Coords, struct2[i].Coords
		els1[i+0*o] = a1[0] - cx1
		els1[i+1*o] = a1[1] - cy1
		els1[i+2*o] = a1[2] - cz1

		els2[i+0*o] = a2[0] - cx2
		els2[i+1*o] = a2[1] - cy2
		els2[i+2*o] = a2[2] - cz2
	}
	X := matrix.MakeDenseMatrix(els1, 3, len(struct1))
	Y := matrix.MakeDenseMatrix(els2, 3, len(struct2))

	// Compute the covariance matrix C = X(Y^T)
	C := must(X.TimesDense(Y.Transpose()))

	// Compute the Singular Value Decomposition of C = VS(W^T)
	//
	// N.B. I suspect that this is the most optimizable portion of this
	// RMSD calculation. If you take a look at the source for SVD (in
	// skelterjohn's go.matrix package), you'll notice that it is a nightmare.
	// I think that if we wrote an optimized implementation specifically for
	// 3x3 matrices, we'd end up squeezing a lot more juice. In particular,
	// we could use the cubic formula to calculate roots of cubic equations
	// in order to find the eigen{values,vectors}.
	V, _, WT, err := C.SVD()
	if err != nil {
		// I'm not quite sure if this is the right thing to do here.
		// It may be that we simply return an invalid RMSD instead.
		panic(err)
	}

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
	var U *matrix.DenseMatrix
	VT := V.Transpose()
	if C.Det() < 0 {
		adjust := matrix.MakeDenseMatrix([]float64{
			1, 0, 0,
			0, 1, 0,
			0, 0, -1,
		}, 3, 3)
		Wadjust := must(WT.TimesDense(adjust))
		U = must(Wadjust.TimesDense(VT))
	} else {
		U = must(WT.TimesDense(VT))
	}

	// Apply the rotational matrix U to X to get the best possible alignment
	// with Y.
	Xbest := must(U.TimesDense(X))

	// Now compute the RMSD between Xbest and Y.
	rmsd := 0.0
	for r := 0; r < 3; r++ {
		for c := 0; c < len(struct1); c++ {
			dist := Xbest.Get(r, c) - Y.Get(r, c)
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
