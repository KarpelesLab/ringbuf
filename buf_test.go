package ringbuf

import (
	"io"
	"testing"
	"time"
)

func TestBuf(t *testing.T) {
	rbuf := make([]byte, 128)

	w, err := New(10)
	if err != nil {
		t.Errorf("failed to initialize buffer")
		return
	}
	w.Write([]byte("hello"))

	r := w.Reader()

	n, err := r.Read(rbuf)
	if n != 5 || err != nil || string(rbuf[:5]) != "hello" {
		t.Errorf("failed simple test, expected to read back hello, got n=%d err=%v", n, err)
	}

	// write more
	w.Write([]byte("helloworld"))

	n, err = r.Read(rbuf)
	if n != 10 || err != nil || string(rbuf[:10]) != "helloworld" {
		t.Errorf("failed buffer reset test, expected to read back helloworld, got n=%d err=%v", n, err)
	}

	r2 := w.Reader()
	r3 := w.Reader()

	// test no new data
	n, err = r.Read(rbuf)
	if err != io.EOF {
		t.Errorf("failed buffer EOF test, expected io.EOF error, got n=%d err=%v", n, err)
	}

	// attempt small read
	n, err = r2.Read(rbuf[:5])
	if n != 5 || err != nil || string(rbuf[:5]) != "hello" {
		t.Errorf("failed buffer read of 5 bytes, expected n=5, got n=%d err=%v", n, err)
	}

	// attempt second small read
	n, err = r2.Read(rbuf[:5])
	if n != 5 || err != nil || string(rbuf[:5]) != "world" {
		t.Errorf("failed buffer read of 5 bytes, expected n=5, got n=%d err=%v", n, err)
	}

	// attempt partial small read
	n, err = r3.Read(rbuf[:7])
	if n != 7 || err != nil || string(rbuf[:7]) != "hellowo" {
		t.Errorf("failed buffer read of 7 bytes, expected n=7, got n=%d err=%v", n, err)
	}

	// write even more (overflow)
	w.Write([]byte("helloworld2"))

	n, err = r.Read(rbuf)
	if err != ErrStaleReader {
		t.Errorf("failed buffer overflow test, expected reader to be invalid, got n=%d err=%v", n, err)
	}

	// testing blocking reader
	w, err = New(64)
	if err != nil {
		t.Errorf("failed to initialize buffer")
		return
	}

	r = w.BlockingReader()
	c := make(chan struct{})
	d := make(chan struct{})

	go func() {
		close(c)
		n, err = r.Read(rbuf[:3])
		close(d)
		r.Close()
	}()
	// make sure we entered the gorouting
	<-c

	time.Sleep(10 * time.Millisecond)

	w.Write([]byte("foo"))
	<-d

	if n != 3 || err != nil || string(rbuf[:3]) != "foo" {
		t.Errorf("failed blocking buffer read of 3 bytes, expected n=3, got n=%d err=%v", n, err)
	}

	// if test hangs there, there's a problem
	w.Close()
}
