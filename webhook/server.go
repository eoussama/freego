package webhook

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// shutdownTimeout bounds how long in-flight deliveries may take to finish
// once the server is asked to stop. A variable for tests.
var shutdownTimeout = 10 * time.Second

// ListenAndServe serves h on path at addr (for example ":8080") until ctx is
// cancelled, then shuts the server down gracefully, giving in-flight
// deliveries up to 10 seconds to finish before their connections are closed
// and their request contexts cancelled. It returns nil after a graceful
// shutdown.
//
// It uses its own [http.ServeMux], so several servers can run in one process.
func ListenAndServe(ctx context.Context, addr, path string, h http.Handler) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return Serve(ctx, l, path, h)
}

// Serve is like [ListenAndServe] on an existing listener, which it closes.
func Serve(ctx context.Context, l net.Listener, path string, h http.Handler) error {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		l.Close()
		return fmt.Errorf("webhook: path %q must start with /", path)
	}
	mux := http.NewServeMux()
	mux.Handle(path, h)

	// Request contexts are cancelled when the connection closes, not when
	// ctx is: in-flight deliveries get the shutdown grace period.
	baseCtx, cancelBase := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelBase()

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		BaseContext:       func(net.Listener) context.Context { return baseCtx },
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(l) }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			// Deliveries still running after the grace period: cancel their
			// contexts and close their connections.
			cancelBase()
			srv.Close()
		}
		if serveErr := <-errc; serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) && err == nil {
			err = serveErr
		}
		return err
	}
}
