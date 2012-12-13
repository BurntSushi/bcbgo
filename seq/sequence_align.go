package seq

import "fmt"

var _ = fmt.Println

type Alignment struct {
	A, B []Residue
}

func newAlignment(length int) Alignment {
	return Alignment{
		A: make([]Residue, 0, length),
		B: make([]Residue, 0, length),
	}
}

func NeedlemanWunsch(A, B []Residue) Alignment {
	// This implementation is taken from the "Needleman–Wunsch_algorithm"
	// Wikipedia article.
	// rows correspond to residues in A
	// cols correspond to residues in B

	// Initialization.
	gapPenalty := getSim62('A', '-')
	matrix := make([][]int, len(A)*len(B))

	// Compute the matrix.
	for i := range A {
		matrix[i] = make([]int, len(B))
		matrix[i][0] = gapPenalty * i
	}
	for j := range B {
		matrix[0][j] = gapPenalty * j
	}
	for i := 1; i < len(A); i++ {
		for j := 1; j < len(B); j++ {
			matrix[i][j] = max3(
				matrix[i-1][j-1]+getSim62(A[i], B[j]),
				matrix[i-1][j]+gapPenalty,
				matrix[i][j-1]+gapPenalty)
		}
	}

	// Now trace an optimal path through the matrix starting at (len(A), len(B))
	aligned := newAlignment(max(len(A), len(B)))
	i, j := len(A)-1, len(B)-1
	for i > 0 && j > 0 {
		s := matrix[i][j]
		sdiag := matrix[i-1][j-1]
		// sup := matrix[i][j-1]
		sleft := matrix[i-1][j]
		switch {
		case s == sdiag+getSim62(A[i], B[j]):
			aligned.A = append(aligned.A, A[i])
			aligned.B = append(aligned.B, B[j])
			i--
			j--
		case s == sleft+gapPenalty:
			aligned.A = append(aligned.A, A[i])
			aligned.B = append(aligned.B, '-')
			i--
		default:
			aligned.A = append(aligned.A, '-')
			aligned.B = append(aligned.B, B[j])
			j--
		}
	}
	if i == 0 || j == 0 {
		aligned.A = append(aligned.A, A[i])
		aligned.B = append(aligned.B, B[j])
	}
	for i > 0 {
		i--
		aligned.A = append(aligned.A, A[i])
		aligned.B = append(aligned.B, '-')
	}
	for j > 0 {
		j--
		aligned.A = append(aligned.A, '-')
		aligned.B = append(aligned.B, B[j])
	}

	// Since we built the alignment in backwards, we must reverse the alignment.
	for i, j := 0, len(aligned.A)-1; i < j; i, j = i+1, j-1 {
		aligned.A[i], aligned.A[j] = aligned.A[j], aligned.A[i]
		aligned.B[i], aligned.B[j] = aligned.B[j], aligned.B[i]
	}

	return aligned
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func max3(a, b, c int) int {
	switch {
	case a > b && a > c:
		return a
	case b > c:
		return b
	}
	return c
}
