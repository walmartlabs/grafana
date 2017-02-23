package models

import (
	"errors"
	"time"

	"github.com/grafana/grafana/pkg/components/simplejson"
)

var (
	ErrDashboardVersionNotFound = errors.New("Dashboard version not found")
	ErrNoVersionsForSlug        = errors.New("No dashboard versions found for the given slug")
)

// A DashboardVersion represents the comparable data in a dashboard, allowing
// diffs of the dashboard to be performed.
type DashboardVersion struct {
	Id          int64
	DashboardId int64
	Slug        string
	Version     int

	Created time.Time

	CreatedBy int64

	Message string
	Data    *simplejson.Json
}

// DashboardVersionDTO represents a dashboard version, without the dashboard
// map.
type DashboardVersionDTO struct {
	Id          int64     `json:"id"`
	DashboardId int64     `json:"dashboardId"`
	Slug        string    `json:"slug"`
	Version     int       `json:"version"`
	Created     time.Time `json:"created"`
	CreatedBy   int64     `json:"createdBy"`
	Message     string    `json:"message"`
}

//
// COMMANDS
//

// GetDashboardVersionCommand contains the data required to execute the
// sqlstore.GetDashboardVersionCommand, which returns the DashboardVersion for
// the given Version.
type GetDashboardVersionCommand struct {
	Slug    string `json:"slug" binding:"Required"`
	Version int    `json:"version" binding:"Required"`

	Result *DashboardVersion
}

// GetDashboardVersionsCommand contains the data required to execute the
// sqlstore.GetDashboardVersionsCommand, which returns all
type GetDashboardVersionsCommand struct {
	Slug string `json:"slug" binding:"Required"`

	Result []*DashboardVersion
}

// RestoreDashboardVersionCommand creates a new dashboard version.
type RestoreDashboardVersionCommand struct {
	Slug    string `json:"slug"`
	Version int    `json:"version" binding:"Required"`

	Result *DashboardVersion
}

// CompareDashboardVersionsCommand is used to compare two versions.
type CompareDashboardVersionsCommand struct {
	Slug     string `json:"slug"`
	Original int    `json:"original" binding:"Required"`
	New      int    `json:"new" binding:"Required"`

	Delta map[string]interface{} `json:"delta"`
}
