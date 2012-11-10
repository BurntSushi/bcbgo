package rmsd

import (
	"fmt"
	"unsafe"

	"github.com/BurntSushi/bcbgo/io/pdb"
)

// #cgo CFLAGS: -O3 -ffast-math
// #include "bridge.h"
// #include "qcprot.h"
import "C"

func CQCRMSD(struct1, struct2 pdb.Atoms) float64 {
	if len(struct1) != len(struct2) {
		panic(fmt.Sprintf("Computing the RMSD of two structures require that "+
			"they have equal length. But the lengths of the two structures "+
			"provided are %d and %d.", len(struct1), len(struct2)))
	}

	cols := len(struct1)
	X := C.MatInit(3, C.int(cols))
	Y := C.MatInit(3, C.int(cols))
	for i := 0; i < cols; i++ {
		xc, yc := struct1[i].Coords, struct2[i].Coords

		C.MatSet(X, C.int(i), C.double(xc[0]), C.double(xc[1]), C.double(xc[2]))
		C.MatSet(Y, C.int(i), C.double(yc[0]), C.double(yc[1]), C.double(yc[2]))
	}

	rot := make([]C.double, 9)
	rmsd := C.CalcRMSDRotationalMatrix(
		X, Y,
		C.int(cols),
		(*C.double)(unsafe.Pointer(&rot[0])),
		(*C.double)(nil))

	C.MatDestroy(X)
	C.MatDestroy(Y)

	return float64(rmsd)
}
