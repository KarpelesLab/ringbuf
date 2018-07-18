package ringbuf

import "testing"

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
	if n != 5 || err != nil {
		t.Errorf("failed simple test, expected to read back hello, got n=%d err=%v", n, err)
	}

	// write more
	w.Write([]byte("helloworld"))

	n, err = r.Read(rbuf)
	if n != 10 || err != nil {
		t.Errorf("failed buffer reset test, expected to read back helloworld, got n=%d err=%v", n, err)
	}

	// write even more (overflow)
	w.Write([]byte("helloworld2"))

	n, err = r.Read(rbuf)
	if err != ErrInvalidReader {
		t.Errorf("failed buffer overflow test, expected reader to be invalid, got n=%d err=%v", n, err)
	}

}
