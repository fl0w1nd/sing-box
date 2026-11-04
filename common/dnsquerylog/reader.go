package dnsquerylog

import (
	"bytes"
	"context"
	"io"
	"os"
)

const (
	readBlockSize = 32 * 1024
	// maxLineSize bounds a line kept while reading backward; a longer one is
	// skipped as broken.
	maxLineSize = 4 * 1024 * 1024
	// cancelCheckLines is how often a read checks whether it was canceled.
	cancelCheckLines = 256
)

// readLinesBackward visits the lines of the first size bytes of the file at
// path, last first, until visit returns false or ctx is done. It reads in
// blocks, so memory holds a block and the line being read rather than the
// file. Empty lines are skipped. The line passed to visit is only valid
// during the call.
func readLinesBackward(ctx context.Context, path string, size int64, visit func(line []byte) bool) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	// The file may have been truncated after a failed write.
	if info, statErr := file.Stat(); statErr == nil {
		size = min(size, info.Size())
	}
	// buffer holds a block followed by the start of the line it continues,
	// read so far: the tail.
	buffer := make([]byte, 2*readBlockSize)
	var (
		tail     int
		skipping bool
		lines    int
	)
	end := size
	for end > 0 {
		n := int(min(int64(readBlockSize), end))
		if len(buffer)-tail < n {
			grown := make([]byte, 2*len(buffer))
			copy(grown[len(grown)-tail:], buffer[len(buffer)-tail:])
			buffer = grown
		}
		start := len(buffer) - tail - n
		_, err = file.ReadAt(buffer[start:start+n], end-int64(n))
		if err != nil && err != io.EOF {
			return err
		}
		end -= int64(n)
		data := buffer[start:]
		for {
			index := bytes.LastIndexByte(data, '\n')
			if index < 0 {
				break
			}
			line := data[index+1:]
			data = data[:index]
			if skipping {
				// The rest of a line too long to keep.
				skipping = false
				continue
			}
			if len(line) == 0 {
				continue
			}
			lines++
			if lines%cancelCheckLines == 0 && ctx.Err() != nil {
				return ctx.Err()
			}
			if !visit(line) {
				return nil
			}
		}
		if end == 0 {
			if len(data) > 0 && !skipping {
				visit(data)
			}
			break
		}
		if len(data) > maxLineSize {
			skipping = true
			tail = 0
			continue
		}
		if skipping {
			tail = 0
			continue
		}
		copy(buffer[len(buffer)-len(data):], data)
		tail = len(data)
	}
	return ctx.Err()
}
