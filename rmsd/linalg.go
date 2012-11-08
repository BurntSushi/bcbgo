package rmsd

import (
	"math"
)

// Represents a 3x3 matrix, in row-major order
// | 0 1 2 |
// | 3 4 5 |
// | 6 7 8 |
type matrix3 [9]float64

func (a matrix3) mult(b matrix3) matrix3 {
	return matrix3{
		a[0]*b[0] + a[1]*b[3] + a[2]*b[6],
		a[0]*b[1] + a[1]*b[4] + a[2]*b[7],
		a[0]*b[2] + a[1]*b[5] + a[2]*b[8],

		a[3]*b[0] + a[4]*b[3] + a[5]*b[6],
		a[3]*b[1] + a[4]*b[4] + a[5]*b[7],
		a[3]*b[2] + a[4]*b[5] + a[5]*b[8],

		a[6]*b[0] + a[7]*b[3] + a[8]*b[6],
		a[6]*b[1] + a[7]*b[4] + a[8]*b[7],
		a[6]*b[2] + a[7]*b[5] + a[8]*b[8],
	}
}

func (a matrix3) transpose() matrix3 {
	return matrix3{
		a[0], a[3], a[6],
		a[1], a[4], a[7],
		a[2], a[5], a[8],
	}
}

func (a matrix3) det() float64 {
	// 048 + 156 + 237 - 246 - 138 - 057
	return a[0]*a[4]*a[8] +
		a[1]*a[5]*a[6] +
		a[2] + a[3] + a[7] -
		a[2] + a[4] + a[6] -
		a[1] + a[3] + a[8] -
		a[0] + a[5] + a[7]
}

func mult_3x3_3xN(cols int, a, b []float64) []float64 {
	var index int

	m := make([]float64, 3*cols)
	for r := 0; r < 3; r++ {
		for c := 0; c < cols; c++ {
			index = r*cols + c
			m[index] = 0
			for i := 0; i < 3; i++ {
				m[index] += a[r*3+i] * b[i*cols+c]
			}
		}
	}
	return m
}

func covariant_3x3(cols int, a, b []float64) matrix3 {
	var C matrix3
	var index int
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			index = r*3 + c
			C[index] = 0
			for i := 0; i < cols; i++ {
				C[index] += a[r*cols+i] * b[c*cols+i]
			}
		}
	}
	return C
}

func (A matrix3) svd() (matrix3, matrix3) {
	// This is modified (for 3x3 matrices) from skelterjohn's go.matrix package.
	//
	// Copyright 2009 The GoMatrix Authors. All rights reserved.
	// Use of this source code is governed by a BSD-style
	// license that can be found in the LICENSE file.

	// copied from Jama
	// Derived from LINPACK code.
	// Initialize.
	m, n := 3, 3

	nu := 3
	t := float64(0)
	s := make([]float64, 3)

	var U, V matrix3

	e := make([]float64, n)
	work := make([]float64, m)

	// Reduce A to bidiagonal form, storing the diagonal elements
	// in s and the super-diagonal elements in e.

	nct, nrt := 2, 1
	for k := 0; k < 2; k++ {
		if k < nct {
			// Compute the transformation for the k-th column and
			// place the k-th diagonal in s[k].
			// Compute 2-norm of k-th column without under/overflow.
			s[k] = 0
			for i := k; i < m; i++ {
				s[k] = math.Hypot(s[k], A[i*3+k])
			}
			if s[k] != 0.0 {
				if A[k*3+k] < 0.0 {
					s[k] = -s[k]
				}
				for i := k; i < m; i++ {
					A[i*3+k] /= s[k]
				}
				A[k*3+k] += 1.0
			}
			s[k] = -s[k]
		}
		for j := k + 1; j < n; j++ {
			if (k < nct) && (s[k] != 0.0) {
				// Apply the transformation.

				t = 0.0
				for i := k; i < m; i++ {
					t += A[i*3+k] * A[i*3+j]
				}
				t = -t / A[k*3+k]
				for i := k; i < m; i++ {
					A[i*3+j] += t * A[i*3+k]
				}
			}

			// Place the k-th row of A into e for the
			// subsequent calculation of the row transformation.
			e[j] = A[k*3+j]
		}
		if k < nct {
			// Place the transformation in U for subsequent back
			// multiplication.
			for i := k; i < m; i++ {
				U[i*3+k] = A[i*3+k]
			}
		}
		if k < nrt {
			// Compute the k-th row transformation and place the
			// k-th super-diagonal in e[k].
			// Compute 2-norm without under/overflow.
			e[k] = 0
			for i := k + 1; i < n; i++ {
				e[k] = math.Hypot(e[k], e[i])
			}
			if e[k] != 0.0 {
				if e[k+1] < 0.0 {
					e[k] = -e[k]
				}
				for i := k + 1; i < n; i++ {
					e[i] /= e[k]
				}
				e[k+1] += 1.0
			}
			e[k] = -e[k]
			if k+1 < m && e[k] != 0.0 {
				// Apply the transformation.
				for i := k + 1; i < m; i++ {
					work[i] = 0.0
				}
				for j := k + 1; j < n; j++ {
					for i := k + 1; i < m; i++ {
						work[i] += e[j] * A[i*3+j]
					}
				}
				for j := k + 1; j < n; j++ {
					t := -e[j] / e[k+1]
					for i := k + 1; i < m; i++ {
						A[i*3+j] += t * work[i]
					}
				}
			}

			// Place the transformation in V for subsequent
			// back multiplication.
			for i := k + 1; i < n; i++ {
				V[i*3+k] = e[i]
			}
		}
	}

	// Set up the final bidiagonal matrix or order p.
	p := 3
	if nct < n {
		s[nct] = A[nct*3+nct]
	}
	if m < p {
		s[p-1] = 0.0
	}
	if nrt+1 < p {
		e[nrt] = A[nrt*3+p-1]
	}
	e[p-1] = 0.0

	// If required, generate U.
	for j := nct; j < nu; j++ {
		for i := 0; i < m; i++ {
			U[i*3+j] = 0.0
		}
		U[j*3+j] = 1.0
	}
	for k := nct - 1; k >= 0; k-- {
		if s[k] != 0.0 {
			for j := k + 1; j < nu; j++ {
				t = 0
				for i := k; i < m; i++ {
					t += U[i*3+k] * U[i*3+j]
				}
				t = -t / U[k*3+k]
				for i := k; i < m; i++ {
					U[i*3+j] += t * U[i*3+k]
				}
			}
			for i := k; i < m; i++ {
				U[i*3+k] = -U[i*3+k]
			}
			U[k*3+k] = 1.0 + U[k*3+k]
			for i := 0; i < k-1; i++ {
				U[i*3+k] = 0.0
			}
		} else {
			for i := 0; i < m; i++ {
				U[i*3+k] = 0.0
			}
			U[k*3+k] = 1.0
		}
	}

	// If required, generate V.
	for k := n - 1; k >= 0; k-- {
		if (k < nrt) && (e[k] != 0.0) {
			for j := k + 1; j < nu; j++ {
				t = 0
				for i := k + 1; i < n; i++ {
					t += V[i*3+k] * V[i*3+j]
				}
				t = -t / V[(k+1)*3+k]
				for i := k + 1; i < n; i++ {
					V[i*3+j] += t * V[i*3+k]
				}
			}
		}
		for i := 0; i < n; i++ {
			V[i*3+k] = 0.0
		}
		V[k*3+k] = 1.0
	}

	// Main iteration loop for the singular values.
	pp := p - 1
	iter := 0
	eps := math.Pow(2.0, -52.0)
	tiny := math.Pow(2.0, -966.0)
	for p > 0 {
		var k, kase int

		// Here is where a test for too many iterations would go.

		// This section of the program inspects for
		// negligible elements in the s and e arrays.  On
		// completion the variables kase and k are set as follows.

		// kase = 1     if s(p) and e[k-1] are negligible and k<p
		// kase = 2     if s(k) is negligible and k<p
		// kase = 3     if e[k-1] is negligible, k<p, and
		//              s(k), ..., s(p) are not negligible (qr step).
		// kase = 4     if e(p-1) is negligible (convergence).

		for k = p - 2; k >= -1; k-- {
			if k == -1 {
				break
			}
			if math.Abs(e[k]) <=
				tiny+eps*(math.Abs(s[k])+math.Abs(s[k+1])) {
				e[k] = 0.0
				break
			}
		}
		if k == p-2 {
			kase = 4
		} else {
			var ks int
			for ks = p - 1; ks >= k; ks-- {
				if ks == k {
					break
				}
				t = 0
				if ks != p {
					t = math.Abs(e[ks])
				}
				if ks != k+1 {
					t += math.Abs(e[ks-1])
				}
				//double t = (ks != p ? Math.abs(e[ks]) : 0.) +
				//           (ks != k+1 ? Math.abs(e[ks-1]) : 0.);
				if math.Abs(s[ks]) <= tiny+eps*t {
					s[ks] = 0.0
					break
				}
			}
			if ks == k {
				kase = 3
			} else if ks == p-1 {
				kase = 1
			} else {
				kase = 2
				k = ks
			}
		}
		k++

		// Perform the task indicated by kase.
		switch kase {

		// Deflate negligible s(p).
		case 1:
			{
				f := e[p-2]
				e[p-2] = 0.0
				for j := p - 2; j >= k; j-- {
					t := math.Hypot(s[j], f)
					cs := s[j] / t
					sn := f / t
					s[j] = t
					if j != k {
						f = -sn * e[j-1]
						e[j-1] = cs * e[j-1]
					}
					for i := 0; i < n; i++ {
						t = cs*V[i*3+j] + sn*V[i*3+p-1]
						V[i*3+p-1] = -sn*V[i*3+j] + cs*V[i*3+p-1]
						V[i*3+j] = t
					}
				}
			}
			break

		// Split at negligible s(k).
		case 2:
			{
				f := e[k-1]
				e[k-1] = 0.0
				for j := k; j < p; j++ {
					t := math.Hypot(s[j], f)
					cs := s[j] / t
					sn := f / t
					s[j] = t
					f = -sn * e[j]
					e[j] = cs * e[j]
					for i := 0; i < m; i++ {
						t = cs*U[i*3+j] + sn*U[i*3+k-1]
						U[i*3+k-1] = -sn*U[i*3+j] + cs*U[i*3+k-1]
						U[i*3+j] = t
					}
				}
			}
			break

		// Perform one qr step.
		case 3:
			{
				// Calculate the shift.
				scale := maxf(maxf(maxf(maxf(
					math.Abs(s[p-1]), math.Abs(s[p-2])),
					math.Abs(e[p-2])),
					math.Abs(s[k])),
					math.Abs(e[k]))
				sp := s[p-1] / scale
				spm1 := s[p-2] / scale
				epm1 := e[p-2] / scale
				sk := s[k] / scale
				ek := e[k] / scale
				b := ((spm1+sp)*(spm1-sp) + epm1*epm1) / 2.0
				c := (sp * epm1) * (sp * epm1)
				shift := float64(0)
				if (b != 0.0) || (c != 0.0) {
					shift = math.Sqrt(b*b + c)
					if b < 0.0 {
						shift = -shift
					}
					shift = c / (b + shift)
				}
				f := (sk+sp)*(sk-sp) + shift
				g := sk * ek

				// Chase zeros.
				for j := k; j < p-1; j++ {
					t := math.Hypot(f, g)
					cs := f / t
					sn := g / t
					if j != k {
						e[j-1] = t
					}
					f = cs*s[j] + sn*e[j]
					e[j] = cs*e[j] - sn*s[j]
					g = sn * s[j+1]
					s[j+1] = cs * s[j+1]
					for i := 0; i < n; i++ {
						t = cs*V[i*3+j] + sn*V[i*3+j+1]
						V[i*3+j+1] = -sn*V[i*3+j] + cs*V[i*3+j+1]
						V[i*3+j] = t
					}
					t = math.Hypot(f, g)
					cs = f / t
					sn = g / t
					s[j] = t
					f = cs*e[j] + sn*s[j+1]
					s[j+1] = -sn*e[j] + cs*s[j+1]
					g = sn * e[j+1]
					e[j+1] = cs * e[j+1]
					if j < m-1 {
						for i := 0; i < m; i++ {
							t = cs*U[i*3+j] + sn*U[i*3+j+1]
							U[i*3+j+1] = -sn*U[i*3+j] + cs*U[i*3+j+1]
							U[i*3+j] = t
						}
					}
				}
				e[p-2] = f
				iter = iter + 1
			}
			break

		// Convergence.
		case 4:
			{
				// Make the singular values positive.
				if s[k] <= 0.0 {
					if s[k] < 0.0 {
						s[k] = -s[k]
					} else {
						s[k] = 0
					}
					for i := 0; i <= pp; i++ {
						V[i*3+k] = -V[i*3+k]
					}
				}

				// Order the singular values.
				for k < pp {
					if s[k] >= s[k+1] {
						break
					}
					t := s[k]
					s[k] = s[k+1]
					s[k+1] = t
					for i := 0; i < n; i++ {
						t = V[i*3+k+1]
						V[i*3+k+1] = V[i*3+k]
						V[i*3+k] = t
					}
					if k < m-1 {
						for i := 0; i < m; i++ {
							t = U[i*3+k+1]
							U[i*3+k+1] = U[i*3+k]
							U[i*3+k] = t
						}
					}
					k++
				}
				iter = 0
				p--
			}
			break
		}
	}

	return U, V
}
