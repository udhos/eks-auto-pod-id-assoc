// Package main implements the tool.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/udhos/boilerplate/boilerplate"
	"github.com/udhos/boilerplate/envconfig"
	"go.yaml.in/yaml/v4"
)

func main() {

	//
	// command-line
	//
	var showVersion bool
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.Parse()

	me := filepath.Base(os.Args[0])

	//
	// version
	//
	{
		v := boilerplate.LongVersion(me + " version=" + version)
		if showVersion {
			fmt.Print(v)
			fmt.Println()
			return
		}
		infof("%s", v)
	}

	env := envconfig.NewSimple(me)
	configFile := env.String("CONFIG_FILE", "config.yaml")
	cfg, err := loadConfigFromFile(configFile)
	if err != nil {
		fatalf("failed to load config: %s: %v", configFile, err)
	}

	{
		data, _ := yaml.Marshal(cfg)
		infof("loaded config:\n%s\n", string(data))
	}

	interval := env.Duration("INTERVAL", time.Minute)
	once := env.Bool("RUN_ONCE", false)
	dry := env.Bool("DRY", true)
	addr := env.String("ADDR", ":8080")
	healthPath := env.String("HEALTH_PATH", "/health")
	metricsPath := env.String("METRICS_PATH", "/metrics")
	metricsNamespace := env.String("METRICS_NAMESPACE", "")
	latencyBucketsSeconds := env.Float64Slice("LATENCY_BUCKETS_SECONDS",
		defaultLatencyBucketsSeconds)

	met := newMetrics(metricsNamespace, latencyBucketsSeconds)

	app := newApplication(cfg, met, newRealClient(me, dry, met))

	app.startServer(addr)

	app.serveHealth(healthPath)

	app.serveMetrics(metricsPath)

	// first run immediately
	app.run()

	if once {
		infof("RUN_ONCE=true, exiting")
		app.stopServer()
		os.Exit(0)
	}

	go func() {
		//
		// main loop
		//

		var needCycle bool

		longTicker := time.NewTicker(interval) // periodic last resort cycle (usually 1min)

		shortTicker := time.NewTicker(2 * time.Second) // coalescing/debouncing timer

		for {
			select {
			case <-longTicker.C: // periodic ticker triggers cycle
				needCycle = true
			case <-app.informerCh: // (unbuffered chan) service account informer triggers cycle
				needCycle = true
			case <-shortTicker.C:
				if needCycle {
					needCycle = false
					app.run() // app.run is blocking
				}
			}
		}
	}()

	gracefulShutdown(app)

	infof("main exiting")
}

var defaultLatencyBucketsSeconds = []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5}

func gracefulShutdown(app *application) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	infof("received signal '%v', initiating shutdown", sig)

	app.stopServer()
}
