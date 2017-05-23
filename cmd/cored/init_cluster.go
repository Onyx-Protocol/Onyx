//+build init_cluster

package main

import "chain/core/config"

/*
This file exposes a build tag to automatically initialize a new
Chain Core cluster on boot. By default a new cored process has no
Chain Core cluster and must be explicitly initialized. Initializing
the cluster automatically will be useful for chain core developer
edition. Users will be able to launch cored and immediately have
a useable Chain Core cluster.
*/

func init() {
	config.BuildConfig.InitCluster = true
}
