// Package pkg provides public libraries and utilities that can be imported by other projects.
//
// This package serves as the public API surface of goUserAPI and contains:
//   - errors: Structured error handling for user operations
//   - utilities: Reusable functions for external projects
//
// All code in this package maintains backward compatibility and follows semantic versioning.
//
// Example usage:
//
//   import "github.com/chybatronik/goUserAPI/pkg/errors"
//
//   userErr := errors.NewUserValidationError("INVALID_FIELD", "Field validation failed")
//   http.Error(w, userErr.Error(), userErr.GetHTTPStatus())
package pkg

// This file serves as the Go package documentation placeholder.
// See README.md for detailed package documentation and examples.