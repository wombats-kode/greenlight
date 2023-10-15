package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	// Initialise a new httprouter instance.
	router := httprouter.New()

	//Convert the notFoundResponse() helper to a http.Handler using the
	// http.HandleFunc() adapter, and then set it as the custom error handler for 404
	// Not Found messages
	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	// Convert the methodNotAllowedResponse() helper to a http.Handler and set it as
	// the custom error handler for 405 Method Not Allowed responses.
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	// Register the relevant methods, URL patterns and handler functions for our
	// endpoints using the HandlerFunc() method.
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.createMovieHandler)
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.showMovieHandler)
	// For 'partial' updates us the PATCH request, rather than PUT.
	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.UpdateMovieHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.deleteMovieHandler)

	// Return the httprouter instance.
	// Wrap the router with the panic recovery middleware
	return app.recoverPanic(router)
}
