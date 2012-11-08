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
// The only major change is the omission of the 'weight' matrix.
// We don't use.

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

	"github.com/BurntSushi/bcbgo/pdb"
)

func QCRMSD(struct1, struct2 pdb.Atoms) float64 {
	if len(struct1) != len(struct2) {
		panic(fmt.Sprintf("Computing the RMSD of two structures require that "+
			"they have equal length. But the lengths of the two structures "+
			"provided are %d and %d.", len(struct1), len(struct2)))
	}

	cols := len(struct1)
	var coords1, coords2 [3][]float64
	for i := 0; i < 3; i++ {
		coords1[i] = make([]float64, cols)
		coords2[i] = make([]float64, cols)
	}
	for i := 0; i < cols; i++ {
		coords1[0][i] = struct1[i].Coords[0]
		coords1[1][i] = struct1[i].Coords[1]
		coords1[2][i] = struct1[i].Coords[2]

		coords2[0][i] = struct2[i].Coords[0]
		coords2[1][i] = struct2[i].Coords[1]
		coords2[2][i] = struct2[i].Coords[2]
	}

	rmsd, _ := calcRMSDRotationalMatrix(coords1, coords2)
	return rmsd
}

func calcRMSDRotationalMatrix(
	coords1, coords2 [3][]float64) (float64, [9]float64) {

	centerCoords(coords1)
	centerCoords(coords2)
	E0, A := innerProduct(coords1, coords2)

	return fastCalcRMSDAndRotation(A, E0, len(coords1[0]))
}

func fastCalcRMSDAndRotation(
	A [9]float64, E0 float64, numCoords int) (float64, [9]float64) {

	// These are some crazy names...
	var Sxx, Sxy, Sxz, Syx, Syy, Syz, Szx, Szy, Szz float64
	var Szz2, Syy2, Sxx2, Sxy2, Syz2, Sxz2, Syx2, Szy2, Szx2 float64
	var SyzSzymSyySzz2, Sxx2Syy2Szz2Syz2Szy2, Sxy2Sxz2Syx2Szx2 float64
	var SxzpSzx, SyzpSzy, SxypSyx, SyzmSzy float64
	var SxzmSzx, SxymSyx, SxxpSyy, SxxmSyy float64
	var C [4]float64
	var mxEigenV float64
	var oldg float64 = 0.0
	var b, a, delta, qsqr float64
	var q1, q2, q3, q4, normq float64
	var a11, a12, a13, a14, a21, a22, a23, a24 float64
	var a31, a32, a33, a34, a41, a42, a43, a44 float64
	var a2, x2, y2, z2 float64
	var xy, az, zx, ay, yz, ax float64
	var a3344_4334, a3244_4234, a3243_4233 float64
	var a3143_4133, a3144_4134, a3142_4132 float64
	var evecprec float64 = 1e-6
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

	rmsd := math.Sqrt(fabs(2.0 * (E0 - mxEigenV) / float64(numCoords)))

	a11 = SxxpSyy + Szz - mxEigenV
	a12 = SyzmSzy
	a13 = -SxzmSzx
	a14 = SxymSyx
	a21 = SyzmSzy
	a22 = SxxmSyy - Szz - mxEigenV
	a23 = SxypSyx
	a24 = SxzpSzx
	a31 = a13
	a32 = a23
	a33 = Syy - Sxx - Szz - mxEigenV
	a34 = SyzpSzy
	a41 = a14
	a42 = a24
	a43 = a34
	a44 = Szz - SxxpSyy - mxEigenV
	a3344_4334 = a33*a44 - a43*a34
	a3244_4234 = a32*a44 - a42*a34
	a3243_4233 = a32*a43 - a42*a33
	a3143_4133 = a31*a43 - a41*a33
	a3144_4134 = a31*a44 - a41*a34
	a3142_4132 = a31*a42 - a41*a32
	q1 = a22*a3344_4334 - a23*a3244_4234 + a24*a3243_4233
	q2 = -a21*a3344_4334 + a23*a3144_4134 - a24*a3143_4133
	q3 = a21*a3244_4234 - a22*a3144_4134 + a24*a3142_4132
	q4 = -a21*a3243_4233 + a22*a3143_4133 - a23*a3142_4132

	qsqr = q1*q1 + q2*q2 + q3*q3 + q4*q4
	if qsqr < evecprec {
		q1 = a12*a3344_4334 - a13*a3244_4234 + a14*a3243_4233
		q2 = -a11*a3344_4334 + a13*a3144_4134 - a14*a3143_4133
		q3 = a11*a3244_4234 - a12*a3144_4134 + a14*a3142_4132
		q4 = -a11*a3243_4233 + a12*a3143_4133 - a13*a3142_4132
		qsqr = q1*q1 + q2*q2 + q3*q3 + q4*q4

		if qsqr < evecprec {
			a1324_1423 := a13*a24 - a14*a23
			a1224_1422 := a12*a24 - a14*a22
			a1223_1322 := a12*a23 - a13*a22
			a1124_1421 := a11*a24 - a14*a21
			a1123_1321 := a11*a23 - a13*a21
			a1122_1221 := a11*a22 - a12*a21

			q1 = a42*a1324_1423 - a43*a1224_1422 + a44*a1223_1322
			q2 = -a41*a1324_1423 + a43*a1124_1421 - a44*a1123_1321
			q3 = a41*a1224_1422 - a42*a1124_1421 + a44*a1122_1221
			q4 = -a41*a1223_1322 + a42*a1123_1321 - a43*a1122_1221
			qsqr = q1*q1 + q2*q2 + q3*q3 + q4*q4

			if qsqr < evecprec {
				q1 = a32*a1324_1423 - a33*a1224_1422 + a34*a1223_1322
				q2 = -a31*a1324_1423 + a33*a1124_1421 - a34*a1123_1321
				q3 = a31*a1224_1422 - a32*a1124_1421 + a34*a1122_1221
				q4 = -a31*a1223_1322 + a32*a1123_1321 - a33*a1122_1221
				qsqr = q1*q1 + q2*q2 + q3*q3 + q4*q4

				if qsqr < evecprec {
					/* if qsqr is too small, return the identity matrix. */
					return 0.0, [9]float64{
						1.0, 0.0, 0.0,
						0.0, 1.0, 0.0,
						0.0, 0.0, 1.0,
					}
				}
			}
		}
	}

	normq = math.Sqrt(qsqr)
	q1 /= normq
	q2 /= normq
	q3 /= normq
	q4 /= normq

	a2 = q1 * q1
	x2 = q2 * q2
	y2 = q3 * q3
	z2 = q4 * q4

	xy = q2 * q3
	az = q1 * q4
	zx = q4 * q2
	ay = q1 * q3
	yz = q3 * q4
	ax = q1 * q2

	var rot [9]float64
	rot[0] = a2 + x2 - y2 - z2
	rot[1] = 2 * (xy + az)
	rot[2] = 2 * (zx - ay)
	rot[3] = 2 * (xy - az)
	rot[4] = a2 - x2 + y2 - z2
	rot[5] = 2 * (yz + ax)
	rot[6] = 2 * (zx + ay)
	rot[7] = 2 * (yz - ax)
	rot[8] = a2 - x2 - y2 + z2

	return rmsd, rot
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
