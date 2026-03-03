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
	infof("%s", boilerplate.LongVersion(me))
	env := envconfig.NewSimple(me)

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

	go func() {
		//
		// main loop
		//

		ticker := time.NewTicker(interval)

		for range ticker.C {
			app.run()
			if once {
				infof("RUN_ONCE=true, exiting")
				break
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
