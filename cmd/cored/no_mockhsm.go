//+build no_mockhsm

package main

import (
	"chain/core"
	"chain/core/blocksigner"
	"chain/core/config"
	"chain/database/pg"
	"errors"
)

func init() {
	config.BuildConfig.MockHSM = false
	enableMockHSM = func(pg.DB) []core.RunOption {
		return nil
	}
	mockHSM = func(pg.DB) (blocksigner.Signer, error) {
		return nil, errors.New("this core is not configured to use mockhsm, must configure block hsm url")
	}
}
