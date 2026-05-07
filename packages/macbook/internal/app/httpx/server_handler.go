package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type ServerHandlerConfig struct {
	Mux           http.Handler
	Wrap          func(http.Handler) http.Handler
	SafeQueryKeys []string
}

type requestLogHandler struct {
	next          http.Handler
	safeQueryKeys []string
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

const shutdownTimeout = 10 * time.Second

func RunServer(addr string, listener net.Listener, server *http.Server) error {
	if listener == nil {
		return errors.New("nil listener")
	}

	if server == nil {
		return errors.New("nil http server")
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- server.Serve(listener)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	defer signal.Stop(sigCh)

	slog.Info("starting server", slog.String("address", addr))

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve http: %w", err)
		}

		return nil
	case sig := <-sigCh:
		slog.Info("shutdown signal received", slog.Any("signal", sig))
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

	defer cancel()

	slog.Info("shutting down server", slog.String("address", addr))

	if err := server.Shutdown(ctx); err != nil {
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, http.ErrServerClosed):
		case errors.Is(err, context.DeadlineExceeded):
			slog.Warn("graceful shutdown timed out, forcing close", slog.String("address", addr))

			if closeErr := server.Close(); closeErr != nil {
				slog.Error("force close server failed", slog.String("address", addr), "error", closeErr)
			}
		default:
			return fmt.Errorf("shutdown server: %w", err)
		}
	}

	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve http: %w", err)
	}

	slog.Info("server stopped", slog.String("address", addr))

	return nil
}

func NewServerHandler(cfg ServerHandlerConfig) http.Handler {
	if cfg.Mux == nil {
		return http.NotFoundHandler()
	}

	handler := cfg.Mux

	if cfg.Wrap != nil {
		handler = cfg.Wrap(handler)
	}

	return requestLogHandler{next: handler, safeQueryKeys: cfg.SafeQueryKeys}
}

func (h requestLogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

	h.next.ServeHTTP(recorder, r)

	status := recorder.status
	attrs := []any{
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"duration_ms", time.Since(started).Milliseconds(),
		"bytes", recorder.bytes,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.UserAgent(),
	}

	if query := safeRequestQuery(r.URL.Query(), h.safeQueryKeys); query != "" {
		attrs = append(attrs, "query", query)
	}

	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		attrs = append(attrs, "forwarded_for", forwardedFor)
	}

	statusClass := strconv.Itoa(status)[0:1] + "xx"

	if status >= http.StatusInternalServerError {
		slog.Error("http request completed", append(attrs, "status_class", statusClass)...)

		return
	}

	if status >= http.StatusBadRequest {
		slog.Warn("http request completed", append(attrs, "status_class", statusClass)...)

		return
	}

	slog.Info("http request completed", attrs...)
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	written, err := r.ResponseWriter.Write(body)
	r.bytes += written

	return written, err
}

func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func safeRequestQuery(values url.Values, keys []string) string {
	safe := url.Values{}

	for _, key := range keys {
		if value, ok := values[key]; ok {
			safe[key] = value
		}
	}

	return safe.Encode()
}
