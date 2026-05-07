package workflowdomain

type eventWriter struct {
	emit     func(string) error
	writeErr error
	written  int
}

func (w *eventWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if w.writeErr != nil {
		return 0, w.writeErr
	}

	if err := w.emit(string(p)); err != nil {
		w.writeErr = err

		return 0, err
	}

	w.written += len(p)

	return len(p), nil
}

func (w *eventWriter) Written() int {
	return w.written
}
