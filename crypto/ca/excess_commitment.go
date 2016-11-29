package ca

import (
	"io"

	"golang.org/x/crypto/sha3"
)

type ExcessCommitment struct {
	Q    Point
	e, s Scalar
}

func (lc *ExcessCommitment) WriteTo(w io.Writer) error {
	err := lc.Q.WriteTo(w)
	if err != nil {
		return err
	}
	_, err = w.Write(lc.e[:])
	if err != nil {
		return err
	}
	_, err = w.Write(lc.s[:])
	return err
}

func (lc *ExcessCommitment) ReadFrom(r io.Reader) error {
	err := lc.Q.readFrom(r)
	if err != nil {
		return err
	}
	_, err = io.ReadFull(r, lc.e[:])
	if err != nil {
		return err
	}
	_, err = io.ReadFull(r, lc.s[:])
	return err
}

func CreateExcessCommitment(q Scalar) (lc ExcessCommitment) {
	lc.Q = multiplyBasePoint(q)
	k := reducedScalar(sha3.Sum512(q[:]))
	R := multiplyBasePoint(k)
	Qbytes := encodePoint(&lc.Q)
	Rbytes := encodePoint(&R)
	eBytes := hash512(Qbytes[:], Rbytes[:])
	lc.e = reducedScalar(eBytes)
	lc.s = multiplyAndAddScalars(q, lc.e, k)
	return lc
}

func (lc ExcessCommitment) Verify() bool {
	negE := negateScalar(lc.e)
	R := multiplyAndAddPoint(negE, lc.Q, lc.s)
	Qbytes := encodePoint(&lc.Q)
	Rbytes := encodePoint(&R)
	ePrimeBytes := hash512(Qbytes[:], Rbytes[:])
	ePrime := reducedScalar(ePrimeBytes)
	return lc.e == ePrime
}
