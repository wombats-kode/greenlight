package main

import (
	"fmt"
	"net/http"
	"time"

	"greenlight.example.com/internal/data"
	"greenlight.example.com/internal/validator"
)

// Add a createMovieHandler for the "POST /v1/movies" endpoint.  For now we simply
// return a plain-text response
func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	// Declare an anonyous struct to hold the information that we expect to be in the
	// HTTP request body.  This struct will be our **target decode destination*.
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	// Use the new readJSON() helper to decode the request body into the input struct.
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Copy the values from the input struct to a new Movie struct.
	movie := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	// Initialise a new Validator instance
	v := validator.New()

	// Call the ValidateMovie() function and return a response containing the errors if
	// any of the checks fail.
	if data.ValidateMovie(v, movie); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Dump the contents of the input struct in a HTTP response
	fmt.Fprintf(w, "%+v\n", input)

}

// Add a showMovieHandler for the "GET /v1/movies/:id" endpoint.  For now we retrieve
// the interpolated id parameter and include it in the response.
func (app *application) showMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Create a new instance of the Movie struct, containing the ID we extracted from
	// the URL and some Dummy data. Note we deliberately haven't set a value for the year
	movie := data.Movie{
		ID:        id,
		CreatedAt: time.Now(),
		Title:     "Casablanca",
		Runtime:   102,
		Genres:    []string{"drama", "romance", "war"},
		Version:   1,
	}

	// Encode the struct to JSON and send it as the HTTP response
	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
