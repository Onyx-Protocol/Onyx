package bc

import (
	"chain/errors"
	"io"
)

func (ti *TxInput) WriteTo(ew *errors.Writer, forHashing bool) {
	ti.writeTo(ew, forHashing)
}

func (to *TxOutput) WriteTo(ew *errors.Writer, forHashing bool) {
	to.writeTo(ew, forHashing)
}

func (tx *TxData) WriteToForHash(w io.Writer, forHashing bool) {
	tx.writeTo(w, forHashing)
}
