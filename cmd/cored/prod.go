//+build prod

package main

import (
	"errors"

	"chain/core"
	"chain/core/blocksigner"
	"chain/database/pg"
)

var prod = true

func resetInDevIfRequested(db pg.DB) {}

func devEnableMockHSM(_ pg.DB) []core.RunOption {
	return nil
}

func devHSM(_ pg.DB) (blocksigner.Signer, error) {
	return nil, errors.New("cannot use mockhsm in production, must configure block hsm url")
}
