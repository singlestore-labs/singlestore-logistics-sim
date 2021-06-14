package simulator

import (
	"os"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
	"gopkg.in/yaml.v2"
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type TopicsConfig struct {
	Brokers []string `yaml:"brokers"`
}

type MetricsConfig struct {
	Port int `yaml:"port"`
}

type NormalDistribution struct {
	Avg    float64 `yaml:"avg"`
	Stddev float64 `yaml:"stddev"`
}

func (n *NormalDistribution) ToDist() *distuv.Normal {
	return &distuv.Normal{
		Mu:    n.Avg,
		Sigma: n.Stddev,
	}
}

type Config struct {
	Verbose int `yaml:"verbose"`

	// SimulatorID must be a unique identifier for this process - if multiple simulators are running, each must have a unique id
	SimulatorID string `yaml:"id"`

	// NumWorkers controls the number of goroutines which will run the simulator
	// set to 0 to use the number of cores on the machine
	NumWorkers int `yaml:"num_workers"`

	// SimInterval determines how fast the simulator runs
	// set to 0 to cause the simulator to run as fast as possible
	SimInterval time.Duration `yaml:"sim_interval"`

	StartTime    time.Time     `yaml:"start_time"`
	TickDuration time.Duration `yaml:"tick_duration"`

	MaxPackages  int `yaml:"max_packages"`
	MaxDelivered int `yaml:"max_delivered"`

	PackagesPerTick NormalDistribution `yaml:"packages_per_tick"`
	HoursAtRest     NormalDistribution `yaml:"hours_at_rest"`

	// ProbabilityExpress measures the probability that a package is shipped express
	// should be a value between 0 and 1
	ProbabilityExpress float64 `yaml:"probability_express"`

	// MinShippingDistanceKM is the minimum distance (km) that we will ship packages
	// (in terms of linear distance between origin and destination)
	MinShippingDistanceKM float64 `yaml:"min_shipping_distance_km"`

	// MinAirFreightDistanceKM is the minimum distance (km) that we will send packages by air
	MinAirFreightDistanceKM float64 `yaml:"min_air_freight_distance_km"`

	// AvgLandSpeedKMPH is the average speed (km/h) for land transportation
	AvgLandSpeedKMPH float64 `yaml:"avg_land_speed_kmph"`

	// AvgAirSpeedKMPH is the average speed (km/h) for air transportation
	AvgAirSpeedKMPH float64 `yaml:"avg_air_speed_kmph"`

	Database DatabaseConfig `yaml:"database"`
	Topics   TopicsConfig   `yaml:"topics"`
	Metrics  MetricsConfig  `yaml:"metrics"`
}

func ParseConfigs(filenames []string) (*Config, error) {
	// initialize with default values
	cfg := Config{
		TickDuration: time.Hour,
	}

	for _, filename := range filenames {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		decoder := yaml.NewDecoder(f)
		err = decoder.Decode(&cfg)
		if err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}
