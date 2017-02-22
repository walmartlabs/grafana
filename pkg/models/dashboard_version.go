package models

import (
	"errors"
	"time"

	"github.com/grafana/grafana/pkg/components/simplejson"
)

var (
	ErrDashboardVersionNotFound = errors.New("Dashboard version not found")
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
	// DashboardId int64 `json:"dashboardId" Binding:"required"`
	Slug    string `json:"slug" Binding:"required"`
	Version int    `json:"version" Binding:"required"`

	Result *DashboardVersion
}

// GetDashboardVersionsCommand contains the data required to execute the
// sqlstore.GetDashboardVersionsCommand, which returns all
type GetDashboardVersionsCommand struct {
	// DashboardId int64 `json:"dashboardId" Binding:"required"`
	Slug string `json:"slug" Binding:"required"`

	Result []*DashboardVersion
}

// RestoreDashboardVersionCommand creates a new dashboard version.
type RestoreDashboardVersionCommand struct {
	// DashboardId int64 `json:"dashboardId" Binding:"required"`
	Slug    string `json:"slug" Binding:"required"`
	Version int    `json:"version" Binding:"required"`

	Result *DashboardVersion
}

// CompareDashboardVersionsCommand is used to compare two versions.
type CompareDashboardVersionsCommand struct {
	Slug     string `json:"slug" Binding:"required"`
	Original int    `json:"original" Binding:"required"`
	New      int    `json:"new" Binding:"required"`

	Delta map[string]interface{} `json:"delta"`
}

// Diff computes a JSON diff.
//
// TODO(ben) this is an okay idea but idk if it's really correct
// Probably better to computer the diff in the services package
func (c *CompareDashboardVersionsCommand) Diff(originalJSON,
	newJSON []byte) error {

	c.Delta = map[string]interface{}{
		"Diff": "Goes here",
	}
	return nil
}
