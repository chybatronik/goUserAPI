// Package types provides shared types for the goUserAPI service
package types

// GetUsersParams represents parameters for GetUsers function
type GetUsersParams struct {
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// GetReportsParams represents parameters for GetReports function (Story 3.1)
type GetReportsParams struct {
	Limit     int
	Offset    int
	StartDate *int64 // Epic 3 default: 0 if nil
	EndDate   *int64 // Epic 3 default: current timestamp if nil
	MinAge    *int   // Epic 3 default: 1 if nil
	MaxAge    *int   // Epic 3 default: 120 if nil
}