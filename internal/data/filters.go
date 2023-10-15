package data

import "greenlight.example.com/internal/validator"

type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafelist []string // Add a SortSafelist field to hold supported sort fields
}

func ValidateFilters(v *validator.Validator, f Filters) {
	// Check that the page and page_size parameters contain sensible values
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.Page <= 100, "page_size", "must be a maximum of 100")

	// Check that the sort parameter matched a value in the safelist
	v.Check(validator.PermittedValue(f.Sort, f.SortSafelist...), "sort", "invalid sort values")

}
