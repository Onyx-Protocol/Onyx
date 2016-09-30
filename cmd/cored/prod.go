//+build prod

package main

import "chain/database/pg"

func resetInDevIfRequested(db pg.DB) {}

func initSchemaInDev(db pg.DB) {}
