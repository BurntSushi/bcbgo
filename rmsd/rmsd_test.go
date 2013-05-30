package rmsd

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/BurntSushi/bcbgo/io/pdb"

	matrix "github.com/skelterjohn/go.matrix"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func ExampleRmsd() {
	// If you add a test, make sure you add a corresponding "RMSD: ..."
	// to the output test at the end of this function.
	tests := [][2][]pdb.Coords{
		{
			{
				atom(-2.803, -15.373, 24.556),
				atom(0.893, -16.062, 25.147),
				atom(1.368, -12.371, 25.885),
				atom(-1.651, -12.153, 28.177),
				atom(-0.440, -15.218, 30.068),
				atom(2.551, -13.273, 31.372),
				atom(0.105, -11.330, 33.567),
			},
			{
				atom(-14.739, -18.673, 15.040),
				atom(-12.473, -15.810, 16.074),
				atom(-14.802, -13.307, 14.408),
				atom(-17.782, -14.852, 16.171),
				atom(-16.124, -14.617, 19.584),
				atom(-15.029, -11.037, 18.902),
				atom(-18.577, -10.001, 17.996),
			},
		},
	}
	for _, test := range tests {
		rms := RMSD(test[0], test[1])
		fmt.Printf("RMSD: %f\n", rms)

		rms = rmsd(test[0], test[1])
		fmt.Printf("RMSD: %f\n", rms)
	}
	// Output:
	// RMSD: 0.719106
	// RMSD: 0.719106
}

func TestCovariant(t *testing.T) {
	cols := 11
	tests1 := randomMatrices(10000, 3, cols)
	tests2 := randomMatrices(10000, 3, cols)
	for i, test1 := range tests1 {
		test2 := tests2[i]

		// Compute our covariant
		tC_ := covariant_3x3(cols, test1, test2)
		tC := tmat(tC_[:])

		// Now compute the "correct" covariant.
		mat1 := matrix.MakeDenseMatrix(test1, 3, cols)
		mat2 := matrix.MakeDenseMatrix(test2, 3, cols)
		aC_, _ := mat1.TimesDense(mat2.Transpose())
		aC := tmat(aC_.Array())

		if !tC.equal(aC) {
			t.Fatalf("The covariant of\n%s\nand\n%s\nis\n%s\nbut we said\n%s\n",
				tmat(test1), tmat(test2), aC, tC)
		}
	}
}

func Test_3x3_times_3xN(t *testing.T) {
	cols := 11
	tests1 := randomMatrices(10000, 3, 3)
	tests2 := randomMatrices(10000, 3, cols)
	for i, test1 := range tests1 {
		test2 := tests2[i]

		// Compute our product.
		tC_ := mult_3x3_3xN(cols, test1, test2)
		tC := tmat(tC_[:])

		// Now compute the "correct" product.
		mat1 := matrix.MakeDenseMatrix(test1, 3, 3)
		mat2 := matrix.MakeDenseMatrix(test2, 3, cols)
		aC_, _ := mat1.TimesDense(mat2)
		aC := tmat(aC_.Array())

		if !tC.equal(aC) {
			t.Fatalf("The product of\n%s\nand\n%s\nis\n%s\nbut we said\n%s\n",
				tmat(test1), tmat(test2), aC, tC)
		}
	}
}

func TestSvd(t *testing.T) {
	tests := randomMatrices3(10000)
	for _, test := range tests {
		// Compute our SVD.
		tU_, tV_ := matrix3(test).svd()
		tU, tV := tmat(tU_[:]), tmat(tV_[:])

		// Now compute the "correct" SVD.
		mat := matrix.MakeDenseMatrix(test[:], 3, 3)
		U, _, V, _ := mat.SVD()
		aU, aV := tmat(U.Array()), tmat(V.Array())

		// Now compare them.
		if !aU.equal(tU) {
			t.Fatalf("With matrix\n%s\nU =\n%s\nbut we said\n%s\n",
				tmat(test[:]), aU, tU)
		}
		if !aV.equal(tV) {
			t.Fatalf("With matrix\n%s\n, V =\n%s\n, but we said\n%s\n",
				tmat(test[:]), aV, tV)
		}
	}
}

func BenchmarkMySvd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		test := matrix3(randomMatrix3())
		b.StartTimer()
		test.svd()
	}
}

func BenchmarkGoMatrixSvd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		test := randomMatrix3()
		mat := matrix.MakeDenseMatrix(test[:], 3, 3)
		b.StartTimer()
		mat.SVD()
	}
}

func BenchmarkMyRmsd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		atoms1 := randomAtoms(11)
		atoms2 := randomAtoms(11)
		b.StartTimer()
		rmsd(atoms1, atoms2)
	}
}

func BenchmarkQCRmsd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		atoms1 := randomAtoms(11)
		atoms2 := randomAtoms(11)
		b.StartTimer()
		RMSD(atoms1, atoms2)
	}
}

func BenchmarkQCRmsdMemory(b *testing.B) {
	mem := NewMemory(11)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		atoms1 := randomAtoms(11)
		atoms2 := randomAtoms(11)
		b.StartTimer()
		RMSDMem(mem, atoms1, atoms2)
	}
}

type tmat []float64

func (m tmat) String() string {
	return fmt.Sprintf(`
|%f  %f  %f|
|%f  %f  %f|
|%f  %f  %f|
`, m[0], m[1], m[2], m[3], m[4], m[5], m[6], m[7], m[8])
}

func (m1 tmat) equal(m2 tmat) bool {
	for i := 0; i < 9; i++ {
		if m1[i] != m2[i] {
			return false
		}
	}
	return true
}

func randomMatrices3(cnt int) [][9]float64 {
	ms := make([][9]float64, cnt)
	for i := 0; i < cnt; i++ {
		ms[i] = randomMatrix3()
	}
	return ms
}

func randomMatrix3() (m [9]float64) {
	for i := 0; i < 9; i++ {
		m[i] = rand.Float64() * float64(rand.Intn(100000))
	}
	return
}

func randomMatrices(cnt, rows, cols int) [][]float64 {
	ms := make([][]float64, cnt)
	for i := 0; i < cnt; i++ {
		ms[i] = randomMatrix(rows, cols)
	}
	return ms
}

func randomMatrix(rows, cols int) (m []float64) {
	m = make([]float64, rows*cols)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			m[r*cols+c] = rand.Float64() * float64(rand.Intn(100000))
		}
	}
	return
}

func randomAtoms(cnt int) []pdb.Coords {
	atoms := make([]pdb.Coords, cnt)
	for i := 0; i < cnt; i++ {
		atoms[i] = randomAtom()
	}
	return atoms
}

func randomAtom() pdb.Coords {
	return atom(
		rand.Float64()*float64(rand.Intn(500)),
		rand.Float64()*float64(rand.Intn(500)),
		rand.Float64()*float64(rand.Intn(500)))
}

func atom(x, y, z float64) pdb.Coords {
	return pdb.Coords{x, y, z}
}
