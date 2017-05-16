package sqlstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-xorm/xorm"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/components/simplejson"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/sqlstore/basic"
	"github.com/grafana/grafana/pkg/services/sqlstore/df"
	jsondiff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
)

func init() {
	bus.AddHandler("sql", CompareDashboardVersionsCommand)
	bus.AddHandler("sql", CompareDashboardVersionsHTMLCommand)
	bus.AddHandler("sql", CompareDashboardVersionsBasicCommand)
	bus.AddHandler("sql", CompareDashboardVersionsTokenCommand)
	bus.AddHandler("sql", GetDashboardVersion)
	bus.AddHandler("sql", GetDashboardVersions)
	bus.AddHandler("sql", RestoreDashboardVersion)
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

	delta, err := diffJSON(original, newDashboard)
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

func CompareDashboardVersionsTokenCommand(cmd *m.CompareDashboardVersionsTokenCommand) error {
	// Find original version
	original, err := getDashboardVersion(cmd.DashboardId, cmd.Original)
	if err != nil {
		return err
	}

	newDashboard, err := getDashboardVersion(cmd.DashboardId, cmd.New)
	if err != nil {
		return err
	}

	delta, err := diffTokens(original, newDashboard)
	if err != nil {
		return err
	}

	// marshal it into JSON
	str, err := json.MarshalIndent(delta, "", "  ")
	if err != nil {
		return err
	}

	cmd.Delta = string(str)
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
	order := ""
	if query.OrderBy != "" {
		order = " desc"
	}
	err := x.In("dashboard_id", query.DashboardId).
		OrderBy(query.OrderBy+order).
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
			RestoredFrom:  cmd.Version,
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

	// TODO(ben) move this to the df package
	format := formatter.NewDeltaFormatter()
	return format.FormatAsJson(diff)
}

// diffJSON computes the diff as human-readable string output, for use in HTML
// templating systems.
func diffJSON(original, newDashboard *m.DashboardVersion) (string, error) {
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

	lineWalker := df.NewBasicWalker()
	jsonFormatter := df.NewAsciiFormatter(result, lineWalker.Walk)

	return jsonFormatter.Format(diff)
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

	// New Basic Diff stuff
	//
	// walkFn
	lineWalker := df.NewBasicWalker()
	jsonFormatter := df.NewAsciiFormatter(result, lineWalker.Walk)
	_, err = jsonFormatter.Format(diff)
	if err != nil {
		return "", err
	}

	str, err := basic.Format(jsonFormatter.Lines)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	fmt.Fprintln(buf, `<div class="basic-diff">`)
	fmt.Fprintln(buf, str)
	fmt.Fprintln(buf, `</div>`)

	return buf.String(), nil
}

// diffTokens computes the diff, returning JSONLine tokens.
func diffTokens(original, newDashboard *m.DashboardVersion) ([]*df.JSONLine, error) {
	diff, err := delta(original, newDashboard)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	originalJSON, err := simplejson.NewFromAny(original).Encode()
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(originalJSON, &result)
	if err != nil {
		return nil, err
	}

	lineWalker := df.NewBasicWalker()
	jsonFormatter := df.NewAsciiFormatter(result, lineWalker.Walk)
	_, err = jsonFormatter.Format(diff)
	if err != nil {
		return nil, err
	}

	return jsonFormatter.Lines, nil
}

type version struct {
	Max int
}

// getMaxVersion returns the highest version number in the `dashboard_version`
// table
func getMaxVersion(sess *xorm.Session, dashboardId int64) (int, error) {
	v := version{}
	has, err := sess.Table("dashboard_version").
		Select("MAX(version) AS max"). // thank you sqlite3 :()
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
