# Simple ring buffer in Go

Useful for stuff like storing logs without memory going overboard.

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
	go io.Copy(os.Stdout, l.BlockingReader())

	// (duplicate this to show anytime the log)

	func dmesg() ([]byte, error) {
		// return up to last 1MB of log entries
		return ioutil.ReadAll(l.Reader())
	}
```
