package main

import "github.com/prometheus/client_golang/prometheus/promhttp"

func (a *application) startMetrics(path string) {

	handler := promhttp.InstrumentMetricHandler(
		a.registry, promhttp.HandlerFor(a.registry, promhttp.HandlerOpts{}))

	a.server.mux.Handle(path, handler)
}
