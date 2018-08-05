package ringbuf

import (
	"errors"
	"io"
	"sync/atomic"
)

type Reader struct {
	w        *Writer
	rPos     int64
	cycle    int64
	block    bool
	autoSkip bool
	closed   *uint64
}

var (
	ErrStaleReader = errors.New("ringbuffer reader is stale (didn't read fast enough - do you need a larger buffer?)")
)

// Read will read data from the ringbuffer to the provided buffer. If no
// new data is available, Read() will either return io.EOF (a later call may
// return new data), or block until data becomes available (if set blocking).
func (r *Reader) Read(p []byte) (int, error) {
	if *r.closed > 0 {
		// you can't read from a reader after calling Close on it
		return 0, io.ErrClosedPipe
	}

	n := int64(len(p))

	r.w.mutex.RLock()
	defer r.w.mutex.RUnlock()

	if r.block {
		for r.cycle == r.w.cycle && r.rPos >= r.w.wPos {
			if r.w.closed {
				r.block = false
				break
			}
			r.w.cond.Wait()
		}
	}

	if r.cycle < r.w.cycle-1 {
		if r.autoSkip {
			// skip missed data, resume as far back as possible
			r.cycle = r.w.cycle - 1
			r.rPos = r.w.wPos
		} else {
			return 0, ErrStaleReader
		}
	}

	if r.cycle == r.w.cycle-1 {
		// remaining bytes in buffer
		if r.w.wPos > r.rPos {
			if r.autoSkip {
				// skip
				r.rPos = r.w.wPos
			} else {
				return 0, ErrStaleReader
			}
		}

		avail := r.w.size - r.w.wPos
		if avail >= n {
			copy(p, r.w.data[r.rPos:r.rPos+n])
			r.rPos += n
			if r.rPos >= r.w.size {
				// reached end of buffer
				r.rPos = 0
				r.cycle += 1
			}
			return int(n), nil
		}

		copy(p, r.w.data[r.rPos:])
		r.rPos = 0
		r.cycle += 1

		nextN, err := r.Read(p[avail:])

		return int(avail) + nextN, err
	}

	if r.cycle != r.w.cycle {
		return 0, errors.New("this should not happen, reader is in the future?")
	}

	// easy
	if r.rPos >= r.w.wPos {
		// > shouldn't happen
		return 0, io.EOF
	}

	avail := r.w.wPos - r.rPos

	if n > avail {
		n = avail
	}

	copy(p, r.w.data[r.rPos:r.rPos+n])
	r.rPos += n
	return int(n), nil
}

func (r *Reader) Close() error {
	if atomic.AddUint64(r.closed, 1) != 1 {
		return nil
	}

	r.w.wg.Done()
	return nil
}

// Reset sets the reader's position after the writer's latest write.
func (r *Reader) Reset() {
	r.w.mutex.RLock()
	defer r.w.mutex.RUnlock()

	r.cycle = r.w.cycle
	r.rPos = r.w.wPos
}

// SetAutoSkip allows enabling auto skip, when this reader hasn't been reading
// fast enough and missed some data. This is generally unsafe, but in some
// cases may be useful to avoid having to handle stale readers.
func (r *Reader) SetAutoSkip(enabled bool) {
	r.autoSkip = enabled
}
