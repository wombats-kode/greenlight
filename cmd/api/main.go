package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

// Define a config stuct to hold all the configuration settings for our applciation.
// We will read in these configurations settings when the application starts.
type config struct {
	port int
	env  string
}

// Define and application struct to hold the dependencies for our HTTP handlers, helpers
// and middleware.
type application struct {
	config config
	logger *slog.Logger
}

func main() {
	// Declare an instance of the config struct
	var cfg config

	// Read the value of the port and env command-line flags into the config struct. We
	// default to using the port number 4000 and the environment 'development' if no
	// corresponding flags are provided
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (Development|staging|production)")
	flag.Parse()

	// Initialise a new structured logger which writes log entries to the standard out
	// stream
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Declare an instance of the application struct, containing the config struct and
	// the logger.
	app := &application{
		config: cfg,
		logger: logger,
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
	err := srv.ListenAndServe()
	logger.Error(err.Error())
	os.Exit(1)

}
