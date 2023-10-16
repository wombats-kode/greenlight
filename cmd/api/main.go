package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	// Note that we alias the import to th blank identifier, to stop Go
	// compiler complaining that the package isnt being used.
	_ "github.com/lib/pq"
	"greenlight.example.com/internal/data"
)

const version = "1.0.0"

// Define a config stuct to hold all the configuration settings for our applciation.
// We will read in these configurations settings when the application starts.
type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  time.Duration
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

// Define and application struct to hold the dependencies for our HTTP handlers, helpers
// and middleware.
type application struct {
	config config
	logger *slog.Logger
	models data.Models
}

func main() {
	// Declare an instance of the config struct
	var cfg config

	// Read the value of the port and env command-line flags into the config struct. We
	// default to using the port number 4000 and the environment 'development' if no
	// corresponding flags are provided
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (Development|staging|production)")

	// Read the DSN value from the db-dsn command line flag into the config struct.
	flag.StringVar(&cfg.db.dsn, "db-dsn", os.Getenv("GREENLIGHT_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")

	// Command-line flags to read the settings for Rate-limiting HTTP requests. Note that
	// we use true as the default for the 'enabled' setting.
	flag.Float64Var(&cfg.limiter.rps, "limiter.rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	flag.Parse()

	// Initialise a new structured logger which writes log entries to the standard out
	// stream
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Call the openDB() helper function to create a connection pool, passing in the
	// config struct. If this returns an error, we log it and exit the application.
	db, err := openDB(cfg)
	if err != nil {
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}
	}
	// Defer a call to db.Close() so that the connection pool is closed before the
	// main() function exits.
	defer db.Close()

	logger.Info("database connection pool established")

	// Declare an instance of the application struct, containing the config struct and
	// the logger.
	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	// Declare a HTTPserver which listens on the port provided in the config struct, uses
	// the servemux we created above as the Handler, has some sensible timeout settings
	// and writes any log messages to the structured logger at Error level.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(), // Use the httprouter instance returned by app.routes()
		IdleTimeout:  time.Minute,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	// Start the HTTP server.
	logger.Info("starting server", "addr", srv.Addr, "env", cfg.env)
	err = srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)

}

// The openDB function returns a sql.DB connection pool
func openDB(cfg config) (*sql.DB, error) {
	// Use sql.Open() to create an emppty connection pool, using the DSN from
	// the config struct.
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	// Set the maximum number of open (in-use + idle) connections in the pool. Note that
	// passing a value less than or equal to 0 will mean there is no limit.
	db.SetMaxOpenConns(cfg.db.maxOpenConns)

	// Set the maximum number of idle connections in the pool. Again, passing a value
	// less than or equal to 0 will mean there is no limit.
	db.SetMaxIdleConns(cfg.db.maxIdleConns)

	// Set the maximum idle timeout for connections in the pool. Passing a duration less
	// than or equal to 0 will mean that connections are not closed due to their idle time.
	db.SetConnMaxIdleTime(cfg.db.maxIdleTime)

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use PingContext() to establish a new connection to the database, passing
	// in the context we created above as a parameter.  If the connection couldn't
	// be established withing the 5 seconds deadlinem then this will return an error.
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	// Return the connection pools
	return db, nil
}
