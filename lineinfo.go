package main

type LineInfo chan string

func (w LineInfo) Write(p []byte) (int, error) {
	if len(p) < 1 {
		return 0, nil
	}

	w <- string(p)
	return len(p), nil
}

func (w LineInfo) Close() error {
	close(w)
	return nil
}
