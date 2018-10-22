package ringbuf

import (
	"errors"
	"io"
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

	closed bool
	mutex  sync.RWMutex
	cond   *sync.Cond
	wg     sync.WaitGroup
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

// Reader returns a new reader positioned at the buffer's oldest available
// position. The reader can be moved to the most recent position by calling
// its Reset() method.
//
// If there isn't enough data to read, the reader's read method will return
// error io.EOF. If you need Read() to not return until new data is available,
// use BlockingReader()
func (w *Writer) Reader() *Reader {
	if w.closed {
		return nil
	}

	cycle := w.cycle
	pos := w.wPos
	// rewind
	if cycle > 0 {
		cycle = cycle - 1
	} else {
		pos = 0
	}

	w.wg.Add(1)

	return &Reader{
		w:      w,
		block:  false,
		cycle:  cycle,
		rPos:   pos,
		closed: new(uint64),
	}
}

// BlockingReader returns a new reader positioned at the buffer's oldest
// available position which reads will block if no new data is available.
func (w *Writer) BlockingReader() *Reader {
	if w.closed {
		return nil
	}

	cycle := w.cycle
	pos := w.wPos
	// rewind
	if cycle > 0 {
		cycle = cycle - 1
	} else {
		pos = 0
	}

	w.wg.Add(1)

	return &Reader{
		w:      w,
		block:  true,
		cycle:  cycle,
		rPos:   pos,
		closed: new(uint64),
	}
}

// Write to buffer, will always succeed
func (w *Writer) Write(buf []byte) (int, error) {
	n := int64(len(buf))

	// lock buffer while writing
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.closed {
		return 0, io.ErrClosedPipe
	}

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
	} else if int64(len(buf)) == remain {
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

// Close will cause all readers to return EOF once they have read the whole
// buffer and will wait until all readers have called Close(). If you do not
// need EOF synchronization you can ignore the whole close system as it is not
// used to free any resources, but if you use ringbuf as an output buffer, for
// example, it will enable waiting for writes to have completed prior to
// ending the program.
//
// Note that if any reader failed to call close prior to end and being freed
// this will cause a deadlock.
func (w *Writer) Close() error {
	w.mutex.Lock()
	if w.closed {
		w.mutex.Unlock()
		// calling close multiple times isn't an error
		return nil
	}
	w.closed = true

	// wake all readers (they will really start moving after the unlock)
	w.cond.Broadcast()

	w.mutex.Unlock()

	// wait for everyone to complete
	w.wg.Wait()
	return nil
}
