// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btclog

import (
	"fmt"

	"github.com/btcsuite/seelog"
)

// Ensure SubsystemLogger implements the Logger interface.
var _ Logger = &SubsystemLogger{}

// SubsystemLogger is a concrete implementation of the Logger interface which
// provides a level-base logger backed by a single seelog instance suitable
// for use by subsystems (and packages) in an overall application.  It allows
// a logger instance per subsystem to be created so each can have its own
// logging level and a prefix which identifies which subsystem the message is
// coming from.  Each instance is backed by a provided seelog instance which
// typically is the same instance for all subsystems, so it doesn't interfere
// with ability to do things create splitters which log to the console and a
// file.
type SubsystemLogger struct {
	closed bool
	log    seelog.LoggerInterface
	level  LogLevel
	prefix string
}

// filter returns whether or not the log message should be filtered based on
// the passed level versus the current level.
func (l *SubsystemLogger) filter(level LogLevel) bool {
	if l.closed || level < l.level {
		return true
	}

	return false
}

// addPrefix prepends the prefix (if any) to the format string and returns it.
func (l *SubsystemLogger) addPrefix(format string) string {
	if l.prefix != "" {
		return l.prefix + format
	}
	return format
}

// addPrefixArg potentially modifies the passed slice of arguments to insert
// the prefix as the first element.
func (l *SubsystemLogger) addPrefixArg(v []interface{}) []interface{} {
	if l.prefix == "" {
		return v
	}

	newV := make([]interface{}, 0, len(v)+1)
	newV = append(newV, l.prefix)
	return append(newV, v...)
}

// Tracef formats message according to format specifier, prepends the prefix (if
// there is one), and writes to log with TraceLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Tracef(format string, params ...interface{}) {
	if l.filter(TraceLvl) {
		return
	}
	l.log.Tracef(l.addPrefix(format), params...)
}

// Debugf formats message according to format specifier, prepends the prefix (if
// there is one), and writes to log with DebugLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Debugf(format string, params ...interface{}) {
	if l.filter(DebugLvl) {
		return
	}
	l.log.Debugf(l.addPrefix(format), params...)
}

// Infof formats message according to format specifier, prepends the prefix (if
// there is one) and writes to log with InfoLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Infof(format string, params ...interface{}) {
	if l.filter(InfoLvl) {
		return
	}
	l.log.Infof(l.addPrefix(format), params...)
}

// Warnf formats message according to format specifier, prepends the prefix (if
// there is one), and writes to log with WarnLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Warnf(format string, params ...interface{}) error {
	if l.filter(WarnLvl) {
		return fmt.Errorf(format, params...)
	}
	return l.log.Warnf(l.addPrefix(format), params...)
}

// Errorf formats message according to format specifier, prepends the prefix (if
// there is one) and writes to log with ErrorLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Errorf(format string, params ...interface{}) error {
	if l.filter(ErrorLvl) {
		return fmt.Errorf(format, params...)
	}
	return l.log.Errorf(l.addPrefix(format), params...)
}

// Criticalf formats message according to format specifier, prepends the prefix
// (if there is one), and writes to log with CriticalLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Criticalf(format string, params ...interface{}) error {
	if l.filter(CriticalLvl) {
		return fmt.Errorf(format, params...)
	}
	return l.log.Criticalf(l.addPrefix(format), params...)
}

// Trace formats message using the default formats for its operands, prepends
// the prefix (if there is one), and writes to log with TraceLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Trace(v ...interface{}) {
	if l.filter(TraceLvl) {
		return
	}
	l.log.Trace(l.addPrefixArg(v)...)
}

// Debug formats message using the default formats for its operands, prepends
// the prefix (if there is one), and writes to log with DebugLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Debug(v ...interface{}) {
	if l.filter(DebugLvl) {
		return
	}
	l.log.Debug(l.addPrefixArg(v)...)
}

// Info formats message using the default formats for its operands, prepends
// the prefix (if there is one), and writes to log with InfoLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Info(v ...interface{}) {
	if l.filter(InfoLvl) {
		return
	}

	l.log.Info(l.addPrefixArg(v)...)
}

// Warn formats message using the default formats for its operands, prepends
// the prefix (if there is one), and writes to log with WarnLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Warn(v ...interface{}) error {
	if l.filter(WarnLvl) {
		return fmt.Errorf(fmt.Sprint(v...))
	}
	return l.log.Warn(l.addPrefixArg(v)...)
}

// Error formats message using the default formats for its operands, prepends
// the prefix (if there is one), and writes to log with ErrorLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Error(v ...interface{}) error {
	if l.filter(ErrorLvl) {
		return fmt.Errorf(fmt.Sprint(v...))
	}
	return l.log.Error(l.addPrefixArg(v)...)
}

// Critical formats message using the default formats for its operands, prepends
// the prefix (if there is one), and writes to log with CriticalLvl.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Critical(v ...interface{}) error {
	if l.filter(CriticalLvl) {
		return fmt.Errorf(fmt.Sprint(v...))
	}
	return l.log.Critical(l.addPrefixArg(v)...)
}

// Level returns the current logging level.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Level() LogLevel {
	return l.level
}

// SetLevel changes the logging level to the passed level.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) SetLevel(level LogLevel) {
	l.level = level
}

// Close closes the subsystem logger so no further messages are logged.  It does
// NOT close the underlying seelog logger as it will likely be used by other
// subsystem loggers.  Closing the underlying seelog logger is the
// responsibility of the caller.
//
// This is part of the Logger interface implementation.
func (l *SubsystemLogger) Close() {
	l.closed = true
}

// NewSubsystemLogger returns a new SubsystemLogger backed by logger with
// prefix before all logged messages at the default log level.  See
// SubsystemLogger for more details.
func NewSubsystemLogger(logger seelog.LoggerInterface, prefix string) Logger {
	return &SubsystemLogger{
		log:    logger,
		prefix: prefix,
		level:  InfoLvl,
	}
}
