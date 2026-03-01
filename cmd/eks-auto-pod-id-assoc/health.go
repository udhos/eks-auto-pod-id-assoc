package main

import (
	"fmt"
	"net/http"
)

func startHealth(addr, path string) {
	infof("registering health route: %s %s",
		addr, path)

	mux := http.NewServeMux()
	server := &http.Server{Addr: addr, Handler: mux}
	mux.HandleFunc(path, func(w http.ResponseWriter,
		_ /*r*/ *http.Request) {
		fmt.Fprintln(w, "200 health ok")
	})

	go func() {
		infof("health server: listening on %s %s",
			addr, path)
		err := server.ListenAndServe()
		fatalf("health server: exited: %v", err)
	}()
}
