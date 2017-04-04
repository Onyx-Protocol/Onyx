//+build disable_tls_only

package main

import "chain/core/config"

func init() {
	config.BuildConfig.TLSOnly = false
}
