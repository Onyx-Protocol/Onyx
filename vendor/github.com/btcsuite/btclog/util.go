// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btclog

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/btcsuite/seelog"
)

// Disabled is a default logger that can be used to disable all logging output.
// The level must not be changed since it's not backed by a real logger.
var Disabled Logger = &SubsystemLogger{level: Off}

// LogLevelFromString returns a LogLevel given a string representation of the
// level along with a boolean that indicates if the provided string could be
// converted.
func LogLevelFromString(level string) (LogLevel, bool) {
	level = strings.ToLower(level)
	for lvl, str := range logLevelStrings {
		if level == str {
			return lvl, true
		}
	}
	return Off, false
}

// NewLoggerFromWriter creates a logger for use with non-btclog based systems.
func NewLoggerFromWriter(w io.Writer, minLevel LogLevel) (Logger, error) {
	l, err := seelog.LoggerFromWriterWithMinLevel(w, seelog.LogLevel(minLevel))
	if err != nil {
		return nil, err
	}

	logger := NewSubsystemLogger(l, "")
	logger.SetLevel(minLevel)

	return logger, nil
}

// NewDefaultBackendLogger returns a new seelog logger with default settings
// that can be used as a backend for SubsystemLoggers.
func NewDefaultBackendLogger() seelog.LoggerInterface {
	config := `
	<seelog type="adaptive" mininterval="2000000" maxinterval="100000000"
		critmsgcount="500" minlevel="trace">
		<outputs formatid="all">
			<console/>
		</outputs>
		<formats>
			<format id="all" format="%Time %Date [%LEV] %Msg%n" />
		</formats>
	</seelog>`

	logger, err := seelog.LoggerFromConfigAsString(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v", err)
		os.Exit(1)
	}

	return logger
}
