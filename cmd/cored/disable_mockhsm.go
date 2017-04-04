//+build disable_mockhsm

package main

import (
	"chain/core"
	"chain/core/blocksigner"
	"chain/core/config"
	"chain/database/pg"
	"errors"
)

func init() {
	config.MockHSM = false
}

func devEnableMockHSM(_ pg.DB) []core.RunOption {
	return nil
}

func devHSM(_ pg.DB) (blocksigner.Signer, error) {
	return nil, errors.New("cannot use mockhsm in production, must configure block hsm url")
}
