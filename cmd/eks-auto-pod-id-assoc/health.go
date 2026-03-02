package main

import (
	"fmt"
	"net/http"
)

func (a *application) serveHealth(path string) {
	infof("registering health route: %s ",
		path)

	a.server.mux.HandleFunc(path, func(w http.ResponseWriter,
		_ /*r*/ *http.Request) {
		fmt.Fprintln(w, "200 health ok")
	})
}
