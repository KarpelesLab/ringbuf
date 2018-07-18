package ringbuf

import (
	"errors"
	"io"
)

type Reader struct {
	w     *Writer
	rPos  int64
	cycle int64
	block bool
}

var (
	ErrInvalidReader = errors.New("buffer reader has become invalid (out of sync with buffer)")
)

func (r *Reader) Read(p []byte) (int, error) {
	n := int64(len(p))

	r.w.mutex.RLock()
	defer r.w.mutex.RUnlock()

	if r.cycle == r.w.cycle {
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

	if r.cycle == r.w.cycle-1 {
		// remaining bytes in buffer
		if r.w.wPos > r.rPos {
			return 0, ErrInvalidReader
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

	return 0, ErrInvalidReader
}

// Reset sets the reader's position after the writer's latest write.
func (r *Reader) Reset() {
	r.w.mutex.RLock()
	defer r.w.mutex.RUnlock()

	r.cycle = r.w.cycle
	r.rPos = r.w.wPos
}
