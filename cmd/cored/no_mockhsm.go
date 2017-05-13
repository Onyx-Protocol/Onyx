//+build no_mockhsm

package main

import (
	"chain/core"
	"chain/core/blocksigner"
	"chain/database/pg"
	"errors"
)

func enableMockHSM(pg.DB) []core.RunOption {
	return nil
}

func mockHSM(pg.DB) (blocksigner.Signer, error) {
	return nil, errors.New("this core is not configured to use mockhsm, must configure block hsm url")
}
