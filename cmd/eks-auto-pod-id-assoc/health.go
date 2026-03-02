package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type httpServer struct {
	mux    *http.ServeMux
	server *http.Server
}

func (a *application) startServer(addr string) {
	mux := http.NewServeMux()
	a.server = httpServer{
		mux:    mux,
		server: &http.Server{Addr: addr, Handler: mux},
	}
	go func() {
		infof("http server listening on %s",
			addr)
		err := a.server.server.ListenAndServe()
		errorf("http server: exited: %v", err)
	}()
}

func (a *application) stopServer() {
	if a.server.server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	if err := a.server.server.Shutdown(ctx); err != nil {
		errorf("http server shutdown error: %v", err)
	}
	a.server.server = nil
}

func (a *application) startHealth(path string) {
	infof("registering health route: %s ",
		path)

	a.server.mux.HandleFunc(path, func(w http.ResponseWriter,
		_ /*r*/ *http.Request) {
		fmt.Fprintln(w, "200 health ok")
	})
}
