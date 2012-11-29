package rmsd

// This is a direct translation from the QCProt code.
// I've done this because cgo doesn't work well with high frequency calls
// from multiple threads.
//
// The only alternative would be to compile the QC C code with the 6c Go
// compiler. But if you do that, you can't use the C standard lib. It's
// definitely doable, but very gruntish.
//
// Note that I've left the structure of the original program in tact so that
// a comparison can be easy. What follows is the original (huge) header.
//
// The only major changes is the omission of the 'weight' matrix, and the
// complete omission of computing the rotational matrix.
// We don't use either.

/*******************************************************************************
 *  -/_|:|_|_\-
 *
 *  File:      qcprot.c
 *  Version:   1.4
 *
 *  Function:  Rapid calculation of the least-squares rotation using a
 *             quaternion-based characteristic polynomial and
 *             a cofactor matrix
 *
 *  Author(s): Douglas L. Theobald
 *             Department of Biochemistry
 *             MS 009
 *             Brandeis University
 *             415 South St
 *             Waltham, MA  02453
 *             USA
 *
 *             dtheobald@brandeis.edu
 *
 *             Pu Liu
 *             Johnson & Johnson Pharmaceutical Research and Development, L.L.C.
 *             665 Stockton Drive
 *             Exton, PA  19341
 *             USA
 *
 *             pliu24@its.jnj.com
 *
 *
 *    If you use this QCP rotation calculation method in a publication, please
 *    reference:
 *
 *      Douglas L. Theobald (2005)
 *      "Rapid calculation of RMSD using a quaternion-based characteristic
 *      polynomial."
 *      Acta Crystallographica A 61(4):478-480.
 *
 *      Pu Liu, Dmitris K. Agrafiotis, and Douglas L. Theobald (2009)
 *      "Fast determination of the optimal rotational matrix for macromolecular
 *      superpositions."
 *      in press, Journal of Computational Chemistry
 *
 *
 *  Copyright (c) 2009-2012 Pu Liu and Douglas L. Theobald
 *  All rights reserved.
 *
 *  Redistribution and use in source and binary forms, with or without
 *  modification, are permitted provided that the following conditions are met:
 *
 *  * Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *  * Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *  * Neither the name of the <ORGANIZATION> nor the names of its contributors
 *    may be used to endorse or promote products derived from this software
 *    without specific prior written permission.
 *
 *  THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 *  "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 *  LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A
 *  PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 *  HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 *  SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 *  LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 *  DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 *  THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 *  (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 *  OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 *  Source:         started anew.
 *
 *  Change History:
 *    2009/04/13      Started source
 *    2010/03/28      Modified FastCalcRMSDAndRotation() to handle tiny qsqr
 *                    If trying all rows of the adjoint still gives too small
 *                    qsqr, then just return identity matrix. (DLT)
 *    2010/06/30      Fixed prob in assigning A[9] = 0 in InnerProduct()
 *                    invalid mem access
 *    2011/02/21      Made CenterCoords use weights
 *    2011/05/02      Finally changed CenterCoords declaration in qcprot.h
 *                    Also changed some functions to static
 *    2011/07/08      put in fabs() to fix taking sqrt of small neg numbers, fp
 *                    error
 *    2012/07/26      minor changes to comments and main.c, more info (v.1.4)
 *
 ******************************************************************************/

import (
	"fmt"
	"math"

	"github.com/BurntSushi/bcbgo/io/pdb"
)

type QcMemory struct {
	coords1, coords2 [3][]float64
}

func NewQcMemory(cols int) QcMemory {
	mem := QcMemory{}
	for i := 0; i < 3; i++ {
		mem.coords1[i] = make([]float64, cols)
		mem.coords2[i] = make([]float64, cols)
	}
	return mem
}

func QCRMSD(struct1, struct2 []pdb.Coords) float64 {
	return QCRMSDMem(NewQcMemory(len(struct1)), struct1, struct2)
}

func QCRMSDMem(mem QcMemory, struct1, struct2 []pdb.Coords) float64 {
	if len(struct1) != len(struct2) {
		panic(fmt.Sprintf("Computing the RMSD of two structures require that "+
			"they have equal length. But the lengths of the two structures "+
			"provided are %d and %d.", len(struct1), len(struct2)))
	}

	cols := len(struct1)
	for i := 0; i < cols; i++ {
		mem.coords1[0][i] = struct1[i].X
		mem.coords1[1][i] = struct1[i].Y
		mem.coords1[2][i] = struct1[i].Z

		mem.coords2[0][i] = struct2[i].X
		mem.coords2[1][i] = struct2[i].Y
		mem.coords2[2][i] = struct2[i].Z
	}
	return calcRMSD(mem.coords1, mem.coords2)
}

func calcRMSD(
	coords1, coords2 [3][]float64) float64 {

	centerCoords(coords1)
	centerCoords(coords2)
	E0, A := innerProduct(coords1, coords2)

	return fastCalcRMSD(A, E0, len(coords1[0]))
}

func fastCalcRMSD(A [9]float64, E0 float64, numCoords int) float64 {
	// These are some crazy names...
	var Sxx, Sxy, Sxz, Syx, Syy, Syz, Szx, Szy, Szz float64
	var Szz2, Syy2, Sxx2, Sxy2, Syz2, Sxz2, Syx2, Szy2, Szx2 float64
	var SyzSzymSyySzz2, Sxx2Syy2Szz2Syz2Szy2, Sxy2Sxz2Syx2Szx2 float64
	var SxzpSzx, SyzpSzy, SxypSyx, SyzmSzy float64
	var SxzmSzx, SxymSyx, SxxpSyy, SxxmSyy float64
	var C [4]float64
	var mxEigenV float64
	var oldg float64 = 0.0
	var b, a, delta float64
	var x2 float64
	var evalprec float64 = 1e-11

	Sxx, Sxy, Sxz = A[0], A[1], A[2]
	Syx, Syy, Syz = A[3], A[4], A[5]
	Szx, Szy, Szz = A[6], A[7], A[8]

	Sxx2 = Sxx * Sxx
	Syy2 = Syy * Syy
	Szz2 = Szz * Szz

	Sxy2 = Sxy * Sxy
	Syz2 = Syz * Syz
	Sxz2 = Sxz * Sxz

	Syx2 = Syx * Syx
	Szy2 = Szy * Szy
	Szx2 = Szx * Szx

	SyzSzymSyySzz2 = 2.0 * (Syz*Szy - Syy*Szz)
	Sxx2Syy2Szz2Syz2Szy2 = Syy2 + Szz2 - Sxx2 + Syz2 + Szy2

	C[2] = -2.0 * (Sxx2 + Syy2 + Szz2 + Sxy2 + Syx2 +
		Sxz2 + Szx2 + Syz2 + Szy2)
	C[1] = 8.0 * (Sxx*Syz*Szy + Syy*Szx*Sxz + Szz*Sxy*Syx -
		Sxx*Syy*Szz - Syz*Szx*Sxy - Szy*Syx*Sxz)

	SxzpSzx = Sxz + Szx
	SyzpSzy = Syz + Szy
	SxypSyx = Sxy + Syx
	SyzmSzy = Syz - Szy
	SxzmSzx = Sxz - Szx
	SxymSyx = Sxy - Syx
	SxxpSyy = Sxx + Syy
	SxxmSyy = Sxx - Syy
	Sxy2Sxz2Syx2Szx2 = Sxy2 + Sxz2 - Syx2 - Szx2

	C[0] = Sxy2Sxz2Syx2Szx2*Sxy2Sxz2Syx2Szx2 +
		(Sxx2Syy2Szz2Syz2Szy2+SyzSzymSyySzz2)*
			(Sxx2Syy2Szz2Syz2Szy2-SyzSzymSyySzz2) +
		(-(SxzpSzx)*(SyzmSzy)+(SxymSyx)*(SxxmSyy-Szz))*
			(-(SxzmSzx)*(SyzpSzy)+(SxymSyx)*(SxxmSyy+Szz)) +
		(-(SxzpSzx)*(SyzpSzy)-(SxypSyx)*(SxxpSyy-Szz))*
			(-(SxzmSzx)*(SyzmSzy)-(SxypSyx)*(SxxpSyy+Szz)) +
		(+(SxypSyx)*(SyzpSzy)+(SxzpSzx)*(SxxmSyy+Szz))*
			(-(SxymSyx)*(SyzmSzy)+(SxzpSzx)*(SxxpSyy+Szz)) +
		(+(SxypSyx)*(SyzmSzy)+(SxzmSzx)*(SxxmSyy-Szz))*
			(-(SxymSyx)*(SyzpSzy)+(SxzmSzx)*(SxxpSyy-Szz))
	mxEigenV = E0
	for i := 0; i < 50; i++ {
		oldg = mxEigenV
		x2 = mxEigenV * mxEigenV
		b = (x2 + C[2]) * mxEigenV
		a = b + C[1]
		delta = (a*mxEigenV + C[0]) / (2.0*x2*mxEigenV + b + a)
		mxEigenV -= delta
		if fabs(mxEigenV-oldg) < fabs(evalprec*mxEigenV) {
			break
		}
	}

	return math.Sqrt(fabs(2.0 * (E0 - mxEigenV) / float64(numCoords)))
}

func innerProduct(coords1, coords2 [3][]float64) (float64, [9]float64) {
	var x1, x2, y1, y2, z1, z2 float64
	numCoords := len(coords1[0])
	fx1, fy1, fz1 := coords1[0], coords1[1], coords1[2]
	fx2, fy2, fz2 := coords2[0], coords2[1], coords2[2]
	var G1, G2 float64 = 0.0, 0.0
	A := [9]float64{
		0, 0, 0,
		0, 0, 0,
		0, 0, 0,
	}
	for i := 0; i < numCoords; i++ {
		x1, y1, z1 = fx1[i], fy1[i], fz1[i]
		x2, y2, z2 = fx2[i], fy2[i], fz2[i]

		G1 += x1*x1 + y1*y1 + z1*z1
		G2 += x2*x2 + y2*y2 + z2*z2

		A[0] += x1 * x2
		A[1] += x1 * y2
		A[2] += x1 * z2

		A[3] += y1 * x2
		A[4] += y1 * y2
		A[5] += y1 * z2

		A[6] += z1 * x2
		A[7] += z1 * y2
		A[8] += z1 * z2
	}
	return 0.5 * (G1 + G2), A
}

func centerCoords(coords [3][]float64) {
	numCoords := len(coords[0])
	var xsum, ysum, zsum float64 = 0.0, 0.0, 0.0
	fx, fy, fz := coords[0], coords[1], coords[2]

	for i := 0; i < numCoords; i++ {
		xsum += fx[i]
		ysum += fy[i]
		zsum += fz[i]
	}
	xsum /= float64(numCoords)
	ysum /= float64(numCoords)
	zsum /= float64(numCoords)
	for i := 0; i < numCoords; i++ {
		fx[i] -= xsum
		fy[i] -= ysum
		fz[i] -= zsum
	}
}

func fabs(a float64) float64 {
	if a >= 0 {
		return a
	}
	return -a
}
