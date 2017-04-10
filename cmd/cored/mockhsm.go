//+build !no_mockhsm

package main

import (
	"chain/core"
	"chain/core/blocksigner"
	"chain/core/mockhsm"
	"chain/database/pg"
)

func enableMockHSM(db pg.DB) []core.RunOption {
	return []core.RunOption{core.MockHSM(mockhsm.New(db))}
}

func mockHSM(db pg.DB) (blocksigner.Signer, error) {
	return mockhsm.New(db), nil
}
