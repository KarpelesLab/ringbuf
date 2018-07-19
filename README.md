# Simple ring buffer in Go

Useful for stuff like storing logs without memory going overboard, or in cases
when you have multiple readers reading a single stream of data from a single
writer.

This code is thread safe and allows either standard readers, or blocking
readers that will wait when no more data is available (this uses
[sync.Cond](https://golang.org/pkg/sync/#Cond) with [RLocker](https://golang.org/pkg/sync/#RWMutex.RLocker)
meaning multiple readers will resume reading in parallel, allowing for using
the most out of goroutines.

Originally this was written as a buffer for [log](https://golang.org/pkg/log/)
allowing to redirect output from logs to multiple targets at the same time
(write to file and stdout at the same time) while also allowing a web API
to fetch the last X MB of entries (as large as the buffer is).

# Documentation

[See the godoc documentation](https://godoc.org/github.com/MagicalTux/ringbuf)

# Sample usage

```go
	l, err := ringbuf.New(1024*1024) // 1MB ring buffer for logs
	if err != nil {
		...
	}
	log.SetOutput(l)

	// output log to stdout
	// (duplicate this to also output to files/etc)
	go io.Copy(os.Stdout, l.BlockingReader())

	func dmesg() ([]byte, error) {
		// return up to last 1MB of log entries
		return ioutil.ReadAll(l.Reader())
	}
```
