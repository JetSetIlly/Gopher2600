package test

// Writer is an implementation of the io.Writer interface. It should be used to
// capture output and to compare with predefined strings.
type Writer struct {
	buffer []byte
}

func (tw *Writer) Write(p []byte) (n int, err error) {
	tw.buffer = append(tw.buffer, p...)
	return len(p), nil
}

// Compare buffered output with predefined/example string
func (tw *Writer) Compare(s string) bool {
	return s == string(tw.buffer)
}
