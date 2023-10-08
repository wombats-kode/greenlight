package main

import (
	"fmt"
	"net/http"
)

// Add a createMovieHandler for the "POST /v1/movies" endpoint.  For now we simply
// return a plain-text response
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "create a new movie")
}

// Add a showMovieHandler for the "GET /v1/movies/:id" endpoint.  For now we retrieve
// the interpolated id parameter and include it in the response.
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	fmt.Fprintf(w, "show the detais of movie %d\n", id)

}
