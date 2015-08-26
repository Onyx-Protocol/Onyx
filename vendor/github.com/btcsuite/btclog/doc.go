// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

/*
Package btclog implements a subsystem aware logger backed by seelog.

Seelog allows you to specify different levels per backend such as console and
file, but it doesn't support levels per subsystem well.  You can create multiple
loggers, but when those are backed by a file, they have to go to different
files.  That is where this package comes in.  It provides a SubsystemLogger
which accepts the backend seelog logger to do the real work.  Each instance of a
SubsystemLogger then allows you specify (and retrieve) an individual level per
subsystem.  All messages are then passed along to the backend seelog logger.
*/
package btclog
