package ringbuf

import (
	"errors"
	"sync"
)

// Ringbuf provides a object suitable for a binary stream written by one source
// and read by multiple consumers at the same time, provided each reader can
// keep up with the writes from the writer (if not, a reader will become
// invalid and will need to be reset)
//
// Note: this is thread safe
type Writer struct {
	data  []byte
	size  int64
	wPos  int64
	cycle int64
	mutex sync.RWMutex
	cond  *sync.Cond
}

func New(size int64) (*Writer, error) {
	if size <= 0 {
		return nil, errors.New("Size must be positive")
	}

	w := &Writer{
		data: make([]byte, size),
		size: size,
	}
	w.cond = sync.NewCond(w.mutex.RLocker())

	return w, nil
}

func (w *Writer) Reader() *Reader {
	cycle := w.cycle
	pos := w.wPos
	if cycle > 0 {
		cycle = cycle - 1
	} else {
		pos = 0
	}

	return &Reader{
		w:     w,
		block: false,
		cycle: cycle,
		rPos:  pos,
	}
}

func (w *Writer) BlockingReader() *Reader {
	cycle := w.cycle
	pos := w.wPos
	if cycle > 0 {
		cycle = cycle - 1
	} else {
		pos = 0
	}

	return &Reader{
		w:     w,
		block: true,
		cycle: cycle,
		rPos:  pos,
	}
}

// Write to buffer, will always succeed
func (w *Writer) Write(buf []byte) (int, error) {
	n := int64(len(buf))

	// lock buffer while writing
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if n > w.size {
		// volume of written data is larger than our buffer (NOTE: will invalidate ALL existing readers)
		cnt := n / w.size
		w.cycle += cnt - 1
		w.wPos += n % w.size
		// only use relevant part of buf
		buf = buf[n-w.size:]
	}

	// copy
	remain := w.size - w.wPos
	copy(w.data[w.wPos:], buf)
	if int64(len(buf)) > remain {
		copy(w.data, buf[remain:])
		w.cycle += 1
	}

	// update cursor position
	w.wPos = ((w.wPos + int64(len(buf))) % w.size)

	w.cond.Broadcast()
	return int(n), nil
}

func (w *Writer) Size() int64 {
	return w.size
}

func (w *Writer) TotalWritten() int64 {
	return w.cycle*w.size + w.wPos
}
