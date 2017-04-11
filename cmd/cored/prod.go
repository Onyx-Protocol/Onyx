//+build prod

package main

import "chain/database/pg"

var prod = true

func resetInDevIfRequested(db pg.DB) {}
