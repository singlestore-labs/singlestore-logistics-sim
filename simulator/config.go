package simulator

import "time"

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type Config struct {
	StartTime    time.Time     `yaml:"startTime`
	TickDuration time.Duration `yaml:"tickDuration"`

	AvgPackagesPerTick    float64 `yaml:"avgPackagesPerTick"`
	StddevPackagesPerTick float64 `yaml:"stddevPackagesPerTick"`

	AvgHoursAtRest    float64 `yaml:"avgHoursAtRest"`
	StddevHoursAtRest float64 `yaml:"stddevHoursAtRest"`

	// ProbabilityExpress measures the probability that a package is shipped express
	// should be a value between 0 and 1
	ProbabilityExpress float64 `yaml:"probabilityExpress"`

	Database DatabaseConfig `yaml:"database"`
}
