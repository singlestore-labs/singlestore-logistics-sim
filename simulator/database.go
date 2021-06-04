package simulator

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	uuid "github.com/satori/go.uuid"

	"simulator/enum"
)

type Database interface {
	Locations() ([]DBLocation, error)
	ActivePackages() ([]DBActivePackage, error)
}

type DBLocation struct {
	LocationID int64
	Kind       enum.LocationKind
	Longitude  float64
	Latitude   float64
}

type DBActivePackage struct {
	PackageID             uuid.UUID
	Method                enum.DeliveryMethod
	DestinationLocationID int64

	// current package position
	Longitude float64
	Latitude  float64

	// the following fields correspond to the most recent transition for this package
	TransitionKind           enum.TransitionKind
	TransitionSeq            int
	TransitionLocationID     int64
	TransitionNextLocationID int64
}

type SingleStore struct {
	db *sqlx.DB
}

var _ Database = &SingleStore{}

func NewSingleStore(config DatabaseConfig) (*SingleStore, error) {
	// We use NewConfig here to set default values. Then we override what we need to.
	mysqlConf := mysql.NewConfig()
	mysqlConf.User = config.Username
	mysqlConf.Passwd = config.Password
	mysqlConf.DBName = config.Database
	mysqlConf.Addr = fmt.Sprintf("%s:%d", config.Host, config.Port)
	mysqlConf.ParseTime = false
	mysqlConf.Timeout = 10 * time.Second
	mysqlConf.InterpolateParams = true
	mysqlConf.AllowNativePasswords = true
	mysqlConf.MultiStatements = false

	mysqlConf.Params = map[string]string{
		"collation_server":    "utf8_general_ci",
		"sql_select_limit":    "18446744073709551615",
		"compile_only":        "false",
		"enable_auto_profile": "false",
		"sql_mode":            "'STRICT_ALL_TABLES'",
	}

	connector, err := mysql.NewConnector(mysqlConf)
	if err != nil {
		return nil, err
	}

	db := sql.OpenDB(connector)

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxIdleConns(20)

	return &SingleStore{db: sqlx.NewDb(db, "mysql")}, nil
}

func (s *SingleStore) Locations() ([]DBLocation, error) {
	out := make([]DBLocation, 0)
	return out, s.db.Select(&out, `
		SELECT
			locationid,
			kind,
			geography_longitude(lonlat) as longitude,
			geography_latitude(lonlat) as latitude
		FROM locations
	`)
}

func (s *SingleStore) ActivePackages() ([]DBActivePackage, error) {
	out := make([]DBActivePackage, 0)
	return out, s.db.Select(&out, `
		SELECT
			p.packageid,
			p.method,
			p.destination_locationid AS destinationlocationid,

			GEOGRAPHY_LONGITUDE(p.lonlat) AS longitude,
			GEOGRAPHY_LATITUDE(p.lonlat) AS latitude,

			pt.kind AS transitionkind,
			pt.seq AS transitionseq,
			pt.locationid AS transitionlocationid,
			pt.next_locationid AS transitionnextlocationid
		FROM packages p
		LEFT JOIN (
			SELECT *, ROW_NUMBER() OVER (PARTITION BY packageid ORDER BY seq desc) AS rownum
			FROM package_transitions
		) pt ON p.packageid = pt.packageid AND rownum = 1;
	`)
}
