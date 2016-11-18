// Package rotation writes and rotates log files.
package rotation

import (
	"bytes"
	"os"
	"strconv"
)

// A File is a log file with associated rotation files.
// The rotation files are named after the base file
// with a numeric suffix: base.1, base.2, and so on.
// Calls to Write write data to the base file.
// When the base file reaches the given size,
// it is renamed to base.1
// (and base.1 is renamed to base.2, and so on)
// and a new base file is opened for subsequent writes.
//
// Any errors encountered while rotating files are ignored.
// Only errors opening and writing the base file are reported.
type File struct {
	base string   // file name
	size int64    // max size of f (limit on w)
	n    int      // number of rotated files
	buf  []byte   // partial line from last write
	f    *os.File // current base file
	w    int64    // bytes written to f
}

// Create creates a log writing to the named file
// with mode 0644 (before umask),
// appending to it if it already exists.
// It will rotate to files name.1, name.2,
// up to name.n.
// The minimum value for n is 1;
// lesser values will be taken as 1.
func Create(name string, size, n int) *File {
	return &File{
		base: name,
		size: int64(size),
		n:    n,
	}
}

var dropmsg = []byte("\nlog write error; some data dropped\n")

// Write writes p to the log file f.
// It writes only complete lines to the underlying file.
// Incomplete lines are buffered in memory
// and written once a NL is encountered.
func (f *File) Write(p []byte) (n int, err error) {
	f.buf = append(f.buf, p...)
	n = len(p)
	if i := bytes.LastIndexByte(f.buf, '\n'); i >= 0 {
		_, err = f.write(f.buf[:i+1])
		// Even if the write failed, discard the entire
		// requested write payload. If we kept it around,
		// then a failure to open the log file would
		// cause us to accumulate data in memory
		// without bound.
		f.buf = f.buf[i+1:]
		if err != nil {
			// If we recover and resume logging,
			// leave a message to indicate we dropped
			// some lines.
			f.buf = append(dropmsg, f.buf...)
		}
	}
	return
}

// write writes the given data to f,
// rotating files if necessary.
func (f *File) write(p []byte) (int, error) {
	// If p would increase the file over the
	// max size, it is time to rotate.
	if f.w+int64(len(p)) > f.size {
		// best-effort; ignore errors
		f.rotate()
		f.f.Close()
		f.f = nil
		f.w = 0
	}
	if f.f == nil {
		var err error
		f.f, err = os.OpenFile(f.base, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644) // #nosec
		if err != nil {
			return 0, err
		}
		f.w, err = f.f.Seek(0, os.SEEK_END)
		if err != nil {
			return 0, err
		}
	}
	n, err := f.f.Write(p)
	f.w += int64(n)
	return n, err
}

func (f *File) rotate() {
	for i := f.n - 1; i > 0; i-- {
		os.Rename(f.name(i), f.name(i+1))
	}
	os.Rename(f.base, f.name(1))
}

func (f *File) name(i int) string {
	return f.base + "." + strconv.Itoa(i)
}
