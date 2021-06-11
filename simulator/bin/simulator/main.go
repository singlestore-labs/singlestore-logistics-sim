package main

import (
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime/pprof"
	"simulator"
	"time"

	"cuelang.org/go/pkg/strconv"
)

type FlagStringSlice []string

func (f *FlagStringSlice) String() string {
	return "[]string"
}

func (f *FlagStringSlice) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())

	configPaths := FlagStringSlice{}
	cpuprofile := ""
	simulatorID := ""

	flag.Var(&configPaths, "config", "path to the config file; can be provided multiple times, files will be merged in the order provided")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to `file`")
	flag.StringVar(&simulatorID, "id", "", "The unique identifier for this simulator process - if multiple simulators are running, each must have a unique id")

	flag.Parse()

	if len(configPaths) == 0 {
		configPaths.Set("config.yaml")
	}

	log.SetFlags(log.Ldate | log.Ltime)

	config, err := simulator.ParseConfigs([]string(configPaths))
	if err != nil {
		log.Fatalf("unable to load config files: %v; error: %+v", configPaths, err)
	}

	// set SimulatorID from env variable
	if sid, ok := os.LookupEnv("SIMULATOR_ID"); ok {
		config.SimulatorID = sid
	}

	// set SimulatorID from flag
	if len(simulatorID) > 0 {
		config.SimulatorID = simulatorID
	}

	// if still empty, fail
	if len(config.SimulatorID) == 0 {
		log.Fatal("simulator id required")
	}

	// set metrics port from env variable
	if mport, ok := os.LookupEnv("METRICS_PORT"); ok {
		metricsPort, err := strconv.ParseInt(mport, 10, 32)
		if err != nil {
			log.Fatalf("unable to parse METRICS_PORT as int: %s; error: %+v", mport, err)
		}
		config.Metrics.Port = int(metricsPort)
	}

	log.Printf("Starting simulator: %s", config.SimulatorID)

	if cpuprofile != "" {
		// disable logging and lower verbosity during profile
		log.SetOutput(ioutil.Discard)
		config.Verbose = 0

		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	go simulator.ExportMetrics(config.Metrics)

	var db simulator.Database
	for {
		db, err = simulator.NewSingleStore(config.Database)
		if err != nil {
			log.Printf("unable to connect to SingleStore: %s; retrying...", err)
			time.Sleep(time.Second)
			continue
		}
		break
	}
	defer db.Close()

	var topics simulator.Topics
	for {
		topics, err = simulator.NewRedpanda(config.Topics)
		if err != nil {
			log.Printf("unable to connect to Redpanda: %s; retrying...", err)
			time.Sleep(time.Second)
			continue
		}
		break
	}
	defer topics.Close()

	if config.StartTime.IsZero() {
		start, err := db.CurrentTime()
		if err != nil {
			log.Fatalf("unable to read current time from SingleStore: %+v", err)
		}
		config.StartTime = start
	}

	log.Printf("starting simulation at %s", config.StartTime)

	locations, err := db.Locations()
	if err != nil {
		log.Fatalf("unable to download locations from SingleStore: %+v", err)
	}
	index, err := simulator.NewLocationIndexFromDB(locations)
	if err != nil {
		log.Fatalf("unable to build location index: %+v", err)
	}

	packages, err := db.ActivePackages(config.SimulatorID)
	if err != nil {
		log.Fatalf("unable to download packages from SingleStore: %+v", err)
	}

	trackers, err := simulator.NewTrackersFromActivePackages(config, index, packages)
	if err != nil {
		log.Fatalf("unable to download locations from SingleStore: %+v", err)
	}

	state := simulator.NewState(config, index, topics, trackers)

	// Trap SIGINT to trigger a shutdown.
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)

	go func() {
		<-signals
		close(state.CloseCh)
	}()

	simulator.Simulate(state)
}
