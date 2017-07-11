//+build no_mockhsm

package main

import (
	"chain/core"
	"chain/core/blocksigner"
	"chain/database/pg"
)

func enableMockHSM(pg.DB) []core.RunOption {
	return nil
}

func mockHSM(pg.DB) blocksigner.Signer {
	return nil
}
