//+build !no_mockhsm

package main

import (
	"chain/core"
	"chain/core/blocksigner"
	"chain/core/config"
	"chain/core/mockhsm"
	"chain/database/pg"
)

func init() {
	config.BuildConfig.MockHSM = true
}

func enableMockHSM(db pg.DB) []core.RunOption {
	return []core.RunOption{core.MockHSM(mockhsm.New(db))}
}

func mockHSM(db pg.DB) blocksigner.Signer {
	return mockhsm.New(db)
}
