package dnsquerylog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/service/filemanager"
)

const segmentExtension = ".jsonl"

// segment is a file of entries in ID order, named after its first ID.
type segment struct {
	firstID uint64
	path    string
	size    int64
	created time.Time
	modTime time.Time
}

func segmentName(firstID uint64) string {
	return strconv.FormatUint(firstID, 10) + segmentExtension
}

func (s *Store) openSegments() error {
	err := filemanager.MkdirAll(s.ctx, s.path, 0o755)
	if err != nil {
		return E.Cause(err, "create DNS query log directory")
	}
	dirEntries, err := os.ReadDir(s.path)
	if err != nil {
		return E.Cause(err, "read DNS query log directory")
	}
	s.fileAccess.Lock()
	defer s.fileAccess.Unlock()
	s.segments = nil
	for _, dirEntry := range dirEntries {
		name := dirEntry.Name()
		if dirEntry.IsDir() || !strings.HasSuffix(name, segmentExtension) {
			continue
		}
		firstID, parseErr := strconv.ParseUint(strings.TrimSuffix(name, segmentExtension), 10, 64)
		if parseErr != nil {
			continue
		}
		info, infoErr := dirEntry.Info()
		if infoErr != nil {
			continue
		}
		s.segments = append(s.segments, &segment{
			firstID: firstID,
			path:    filepath.Join(s.path, name),
			size:    info.Size(),
			created: info.ModTime(),
			modTime: info.ModTime(),
		})
	}
	slices.SortFunc(s.segments, func(a, b *segment) int {
		switch {
		case a.firstID < b.firstID:
			return -1
		case a.firstID > b.firstID:
			return 1
		default:
			return 0
		}
	})
	// Continue the IDs of the newest segment.
	if len(s.segments) > 0 {
		lastID := s.segments[len(s.segments)-1].firstID
		_ = readSegment(s.segments[len(s.segments)-1].path, func(entry Entry) bool {
			lastID = max(lastID, entry.ID)
			return true
		})
		if s.idCounter.Load() < lastID {
			s.idCounter.Store(lastID)
		}
	}
	s.cleanupLocked(time.Now())
	return nil
}

func (s *Store) loopFlush() {
	defer close(s.closed)
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-s.flushSignal:
		case <-s.closing:
			return
		}
		err := s.flush()
		if err != nil {
			s.logger.Error(E.Cause(err, "write DNS query log"))
		}
	}
}

func (s *Store) flush() error {
	s.fileAccess.Lock()
	defer s.fileAccess.Unlock()
	s.access.Lock()
	pending := s.pending
	s.pending = nil
	s.access.Unlock()
	written, err := s.writeLocked(pending)
	if err != nil {
		// Keep the entries not written for the next flush, within the memory
		// size.
		s.access.Lock()
		s.pending = append(pending[written:], s.pending...)
		if overflow := len(s.pending) - s.memorySize*4; overflow > 0 {
			s.pending = s.pending[overflow:]
		}
		s.access.Unlock()
		return err
	}
	// Also when idle, so that old entries go once the retention is over.
	s.cleanupLocked(time.Now())
	return nil
}

// writeLocked appends entries to the segments and returns how many were
// written, so that a failed flush retries only the rest.
func (s *Store) writeLocked(entries []Entry) (int, error) {
	var buffer bytes.Buffer
	now := time.Now()
	var total int
	for total < len(entries) {
		remaining := entries[total:]
		var current *segment
		if s.current != nil && len(s.segments) > 0 {
			current = s.segments[len(s.segments)-1]
			if current.size >= s.segmentSize || now.Sub(current.created) >= segmentDuration {
				err := s.current.Close()
				s.current = nil
				current = nil
				if err != nil {
					return total, err
				}
			}
		}
		if current == nil {
			path := filepath.Join(s.path, segmentName(remaining[0].ID))
			file, err := filemanager.OpenFile(s.ctx, path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return total, err
			}
			s.current = file
			if last := len(s.segments) - 1; last >= 0 && s.segments[last].path == path {
				// Reopened after a failed write.
				current = s.segments[last]
			} else {
				current = &segment{firstID: remaining[0].ID, path: path, created: now}
				s.segments = append(s.segments, current)
			}
		}
		buffer.Reset()
		var written int
		for _, entry := range remaining {
			content, err := json.Marshal(entry)
			if err != nil {
				return total, err
			}
			buffer.Write(content)
			buffer.WriteByte('\n')
			written++
			if current.size+int64(buffer.Len()) >= s.segmentSize {
				break
			}
		}
		n, err := s.current.Write(buffer.Bytes())
		if err != nil {
			s.discardPartialWriteLocked(current, n)
			return total, err
		}
		current.size += int64(buffer.Len())
		current.modTime = now
		total += written
	}
	return total, nil
}

// discardPartialWriteLocked removes what a failed write of n bytes left in the
// current segment, so that the retry does not duplicate entries or continue a
// broken line.
func (s *Store) discardPartialWriteLocked(current *segment, n int) {
	if n <= 0 {
		return
	}
	if s.current.Truncate(current.size) == nil {
		return
	}
	// The broken line is skipped when read. Count it, and write the retry to
	// a new file when the entries go to another segment.
	current.size += int64(n)
	s.current.Close()
	s.current = nil
}

// cleanupLocked removes segments older than the retention, then the oldest
// segments until the total size fits. The current segment is closed when it
// is removed.
func (s *Store) cleanupLocked(now time.Time) {
	var kept []*segment
	var totalSize int64
	for index, item := range s.segments {
		if now.Sub(item.modTime) > s.retention {
			if s.current != nil && index == len(s.segments)-1 {
				s.current.Close()
				s.current = nil
			}
			s.removeSegment(item)
			continue
		}
		kept = append(kept, item)
		totalSize += item.size
	}
	s.segments = kept
	for totalSize > s.maxSize && len(s.segments) > 1 {
		totalSize -= s.segments[0].size
		s.removeSegment(s.segments[0])
		s.segments = s.segments[1:]
	}
}

func (s *Store) removeSegment(item *segment) {
	err := os.Remove(item.path)
	if err != nil && !os.IsNotExist(err) {
		s.logger.Warn("remove DNS query log segment: ", err)
	}
}

func (s *Store) removeAllSegmentsLocked() error {
	if s.current != nil {
		s.current.Close()
		s.current = nil
	}
	var lastErr error
	for _, item := range s.segments {
		err := os.Remove(item.path)
		if err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
	}
	s.segments = nil
	return lastErr
}

// queryFiles walks pending entries and then segments, newest first, until
// collect returns false.
func (s *Store) queryFiles(filter Filter, before uint64, collect func(entry Entry) bool) error {
	s.fileAccess.Lock()
	defer s.fileAccess.Unlock()
	s.access.Lock()
	pending := slices.Clone(s.pending)
	s.access.Unlock()
	// Entries are stored in the order they finished, but Time is when they
	// started, so collect filters each entry by time and only whole segments
	// are skipped.
	for _, entry := range slices.Backward(pending) {
		if !collect(entry) {
			return nil
		}
	}
	var entries []Entry
	for _, item := range slices.Backward(s.segments) {
		if before != 0 && item.firstID >= before {
			continue
		}
		// Every entry of a segment started before its last write.
		if filter.Since > 0 && item.modTime.UnixMilli() < filter.Since {
			return nil
		}
		entries = entries[:0]
		err := readSegment(item.path, func(entry Entry) bool {
			entries = append(entries, entry)
			return true
		})
		if err != nil {
			return err
		}
		for _, entry := range slices.Backward(entries) {
			if !collect(entry) {
				return nil
			}
		}
	}
	return nil
}

// readSegment decodes the entries of a segment, skipping lines left
// incomplete by a crash.
func readSegment(path string, visit func(entry Entry) bool) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var entry Entry
		if json.Unmarshal(scanner.Bytes(), &entry) != nil {
			continue
		}
		if !visit(entry) {
			return nil
		}
	}
	return scanner.Err()
}
