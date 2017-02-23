package sqlstore

import (
	"github.com/go-xorm/xorm"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/components/simplejson"
	m "github.com/grafana/grafana/pkg/models"
	jsondiff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

func init() {
	bus.AddHandler("sql", CompareDashboardVersionsCommand)
	bus.AddHandler("sql", GetDashboardVersion)
	bus.AddHandler("sql", GetDashboardVersions)
	bus.AddHandler("sql", RestoreDashboardVersion)

	// bus.AddHandler("sql", RestoreDeletedDashboard)
	// bus.AddHandler("sql", Blame)
}

// CompareDashboardVersionsCommand computes the JSON diff of two versions,
// assigning the delta of the diff to the `Delta` field.
func CompareDashboardVersionsCommand(cmd *m.CompareDashboardVersionsCommand) error {
	// Find original version
	original, err := getDashboardVersion(cmd.DashboardId, cmd.Original)
	if err != nil {
		return err
	}

	newDashboard, err := getDashboardVersion(cmd.DashboardId, cmd.New)
	if err != nil {
		return err
	}

	delta, err := diff(original, newDashboard)
	if err != nil {
		return err
	}

	cmd.Delta = delta
	return nil
}

// GetDashboardVersion gets the dashboard version for the given dashboard ID
// and version number.
func GetDashboardVersion(query *m.GetDashboardVersionCommand) error {
	result, err := getDashboardVersion(query.DashboardId, query.Version)
	if err != nil {
		return err
	}

	query.Result = result
	return nil
}

// GetDashboardVersions gets all dashboard versions for the given dashboard ID.
func GetDashboardVersions(query *m.GetDashboardVersionsCommand) error {
	err := x.In("dashboard_id", query.DashboardId).Find(&query.Result)
	if err != nil {
		return err
	}

	if len(query.Result) < 1 {
		return m.ErrNoVersionsForDashboardId
	}
	return nil
}

// RestoreDashboardVersion restores the dashboard data to the given version.
func RestoreDashboardVersion(cmd *m.RestoreDashboardVersionCommand) error {
	return inTransaction(func(sess *xorm.Session) error {
		// Check if dashboard version exists in dashboard_version table
		dashboardVersion, err := getDashboardVersion(cmd.DashboardId, cmd.Version)
		if err != nil {
			return err
		}

		dashboard, err := getDashboard(cmd.DashboardId)
		if err != nil {
			return err
		}

		// Update dasboard model
		dashboard.Version = dashboardVersion.Version
		dashboard.Data = dashboardVersion.Data

		rows, err := sess.Id(dashboard.Id).Update(dashboard)
		if err != nil {
			return err
		}
		if rows == 0 {
			return m.ErrDashboardNotFound
		}

		return nil
	})
}

// func RestoreDeletedDashboard(cmd *m.) error {

// }

// func Blame(cmd *m.) error {

// }

// getDashboardVersion is a helper function that gets the dashboard version for
// the given dashboard ID and version ID.
func getDashboardVersion(dashboardId int64, version int) (*m.DashboardVersion, error) {
	dashboardVersions := make([]*m.DashboardVersion, 0)
	err := x.Where("dashboard_id=? AND version=?", dashboardId, version).Find(&dashboardVersions)
	if err != nil {
		return nil, err
	}
	if len(dashboardVersions) < 1 {
		return nil, m.ErrDashboardVersionNotFound
	}
	return dashboardVersions[0], nil
}

// getDashboard gets a dashboard by ID. Used for retrieving the dashboard
// associated with dashboard versions.
func getDashboard(dashboardId int64) (*m.Dashboard, error) {
	dashboard := m.Dashboard{Id: dashboardId}
	has, err := x.Get(&dashboard)
	if err != nil {
		return nil, err
	}
	if has == false {
		return nil, m.ErrDashboardNotFound
	}
	return &dashboard, nil
}

// diff calculates the diff of two JSON objects
func diff(original, newDashboard *m.DashboardVersion) (map[string]interface{}, error) {
	originalJSON, err := simplejson.NewFromAny(original).Encode()
	if err != nil {
		return nil, err
	}

	newJSON, err := simplejson.NewFromAny(newDashboard).Encode()
	if err != nil {
		return nil, err
	}

	differ := jsondiff.New()
	diff, err := differ.Compare(originalJSON, newJSON)
	if err != nil {
		return nil, err
	}

	// TODO(ben) should error with appropriate message
	if !diff.Modified() {
		return nil, nil
	}

	format := formatter.NewDeltaFormatter()
	return format.FormatAsJson(diff)
}
