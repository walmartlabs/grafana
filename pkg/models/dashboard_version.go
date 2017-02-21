package models

import (
	"time"

	"github.com/grafana/grafana/pkg/components/simplejson"
)

// A DashboardVersion represents the comparable data in a dashboard, allowing
// diffs of the dashboard to be performed.
type DashboardVersion struct {
	Id      int64
	Slug    string
	Version int // fk for Dashboard struct

	Created time.Time

	CreatedBy int64

	Message string
	Data    *simplejson.Json
}

// DashboardVersionDTO represents a dashboard version, without the dashboard
// map.
type DashboardVersionDTO struct {
	Id        int64     `json:"id"`
	Slug      string    `json:"slug"`
	Version   int       `json:"version"`
	Created   time.Time `json:"created"`
	CreatedBy int64     `json:"createdBy"`
	Message   string    `json:"message"`
}

// -----------------
// COMMANDS

// CreateDashboardVersionCommand creates a new dashboard version.
type CreateDashboardVersionCommand struct {
}

// GetDashboardVersionsQuery accepts two dashboard versions, and returns the
// diff output of those versions.
type GetDashboardVersionsQuery struct {
	VersionA int
	VersionB int

	Result []*DashboardSnapshot
}
