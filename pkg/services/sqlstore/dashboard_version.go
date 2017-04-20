package sqlstore

import (
	"encoding/json"
	"time"

	"github.com/go-xorm/xorm"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/components/simplejson"
	m "github.com/grafana/grafana/pkg/models"
	diffformatter "github.com/grafana/grafana/pkg/services/sqlstore/formatter"
	jsondiff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"

	xx "github.com/grafana/grafana/pkg/services/sqlstore/df"
)

func init() {
	bus.AddHandler("sql", CompareDashboardVersionsCommand)
	bus.AddHandler("sql", CompareDashboardVersionsHTMLCommand)
	bus.AddHandler("sql", CompareDashboardVersionsBasicCommand)
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

// CompareDashboardVersionsHTMLCommand computes the JSON diff of two versions,
// assigning the delta of the diff to the `Delta` field.
func CompareDashboardVersionsHTMLCommand(cmd *m.CompareDashboardVersionsHTMLCommand) error {
	// Find original version
	original, err := getDashboardVersion(cmd.DashboardId, cmd.Original)
	if err != nil {
		return err
	}

	newDashboard, err := getDashboardVersion(cmd.DashboardId, cmd.New)
	if err != nil {
		return err
	}

	delta, err := diffHTML(original, newDashboard)
	if err != nil {
		return err
	}

	cmd.Delta = delta
	return nil
}

// CompareDashboardVersionsBasicCommand computes the JSON diff of two versions,
// assigning the delta of the diff to the `Delta` field.
func CompareDashboardVersionsBasicCommand(cmd *m.CompareDashboardVersionsBasicCommand) error {
	// Find original version
	original, err := getDashboardVersion(cmd.DashboardId, cmd.Original)
	if err != nil {
		return err
	}

	newDashboard, err := getDashboardVersion(cmd.DashboardId, cmd.New)
	if err != nil {
		return err
	}

	delta, err := diffBasic(original, newDashboard)
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
	err := x.In("dashboard_id", query.DashboardId).
		OrderBy(query.OrderBy).
		Limit(query.Limit, query.Start).
		Find(&query.Result)
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

		version, err := getMaxVersion(sess, dashboard.Id)
		if err != nil {
			return err
		}

		// revert and save to a new dashboard version
		dashboard.Data = dashboardVersion.Data
		dashboard.Updated = time.Now()
		dashboard.UpdatedBy = cmd.UserId
		dashboard.Version = version
		dashboard.Data.Set("version", dashboard.Version)
		// TODO(ben): decide when this should be cleared, or if it should exist at all
		dashboard.Data.Set("restoredFrom", cmd.Version)
		affectedRows, err := sess.Id(dashboard.Id).Update(dashboard)
		if err != nil {
			return err
		}
		if affectedRows == 0 {
			return m.ErrDashboardNotFound
		}

		// save that version a new version
		dashVersion := &m.DashboardVersion{
			DashboardId:   dashboard.Id,
			ParentVersion: cmd.Version,
			Version:       dashboard.Version,
			Created:       time.Now(),
			CreatedBy:     dashboard.UpdatedBy,
			Message:       "",
			Data:          dashboard.Data,
		}
		affectedRows, err = sess.Insert(dashVersion)
		if err != nil {
			return err
		}
		if affectedRows == 0 {
			return m.ErrDashboardNotFound
		}

		cmd.Result = dashboard
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
	dashboardVersion := m.DashboardVersion{}
	has, err := x.Where("dashboard_id=? AND version=?", dashboardId, version).Get(&dashboardVersion)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, m.ErrDashboardVersionNotFound
	}

	dashboardVersion.Data.Set("id", dashboardVersion.DashboardId)
	return &dashboardVersion, nil
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

func delta(original, newDashboard *m.DashboardVersion) (jsondiff.Diff, error) {
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

	if !diff.Modified() {
		return nil, nil
	}

	return diff, nil
}

// diff calculates the diff of two JSON objects. A the two objects are the
// same, the error, as well as the diff, will be nil, indicating that the diff
// algorithm ran successfully but no changes were detected.
func diff(original, newDashboard *m.DashboardVersion) (map[string]interface{}, error) {
	diff, err := delta(original, newDashboard)
	if err != nil {
		return nil, err
	}

	format := formatter.NewDeltaFormatter()
	return format.FormatAsJson(diff)
}

// diffHTML computes the diff as human-readable string output, for use in HTML
// templating systems.
func diffHTML(original, newDashboard *m.DashboardVersion) (string, error) {
	diff, err := delta(original, newDashboard)
	if err != nil {
		return "", err
	}

	result := make(map[string]interface{})
	originalJSON, err := simplejson.NewFromAny(original).Encode()
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(originalJSON, &result)
	if err != nil {
		return "", err
	}

	format := diffformatter.NewAsciiFormatter(result, diffformatter.AsciiFormatterConfig{
		ShowArrayIndex: false,
		Coloring:       true,
	})
	pretty, err := format.Format(diff)
	if err != nil {
		return "", err
	}

	return `<pre><code>` + pretty + `</pre></code>`, nil
}

// diffBasic computes the diff as human-readable string output, for use in HTML
// templating systems.
func diffBasic(original, newDashboard *m.DashboardVersion) (string, error) {
	diff, err := delta(original, newDashboard)
	if err != nil {
		return "", err
	}

	result := make(map[string]interface{})
	originalJSON, err := simplejson.NewFromAny(original).Encode()
	if err != nil {
		return "", err
	}
	err = json.Unmarshal(originalJSON, &result)
	if err != nil {
		return "", err
	}

	format := xx.NewBasicFormatter(result)
	return format.Format(diff)
}

type version struct {
	Max int
}

// getMaxVersion returns the highest version number in the `dashboard_version`
// table
func getMaxVersion(sess *xorm.Session, dashboardId int64) (int, error) {
	v := version{}
	has, err := sess.Table("dashboard_version").
		Select("MAX(version) AS max").
		Where("dashboard_id = ?", dashboardId).
		Get(&v)
	if !has {
		return 0, m.ErrDashboardNotFound
	}
	if err != nil {
		return 0, err
	}

	v.Max++
	return v.Max, nil
}

// TODO(ben): move all the diff stuff to it's own package
