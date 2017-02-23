package sqlstore

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/grafana/grafana/pkg/components/simplejson"
	m "github.com/grafana/grafana/pkg/models"
)

func updateTestDashboard(dashboard *m.Dashboard, data map[string]interface{}) {
	data["title"] = dashboard.Title

	saveCmd := m.SaveDashboardCommand{
		OrgId:     dashboard.OrgId,
		Overwrite: true,
		Dashboard: simplejson.NewFromAny(data),
	}

	err := SaveDashboard(&saveCmd)
	So(err, ShouldBeNil)
}

func TestGetDashboardVersion(t *testing.T) {
	Convey("Testing dashboard version retrieval", t, func() {
		InitTestDB(t)

		Convey("Get a version slug and version ID", func() {
			savedDash := insertTestDashboard("test dash 26", 1, "diff")

			cmd := m.GetDashboardVersionCommand{
				Slug:    savedDash.Slug,
				Version: savedDash.Version,
			}

			err := GetDashboardVersion(&cmd)
			So(err, ShouldBeNil)
			So(savedDash.Slug, ShouldEqual, cmd.Slug)
			So(savedDash.Version, ShouldEqual, cmd.Version)

			// This won't pass until we add referential integrity -- the result
			// has a version with the type `json.Number` but the saved dashboard
			// has a version with the type `int`, causing the DeepEqual to
			// return false.

			// eq := reflect.DeepEqual(savedDash.Data, cmd.Result.Data)
			// So(eq, ShouldEqual, true)
		})

		Convey("Attempt to get a version that doesn't exist", func() {
			cmd := m.GetDashboardVersionCommand{
				Slug:    "not-existent-slug",
				Version: 123,
			}

			err := GetDashboardVersion(&cmd)
			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, m.ErrDashboardVersionNotFound)
		})
	})
}

func TestGetDashboardVersions(t *testing.T) {
	Convey("Testing dashboard versions retrieval", t, func() {
		InitTestDB(t)
		savedDash := insertTestDashboard("test dash 43", 1, "diff-all")

		Convey("Get all versions for a given slug", func() {
			cmd := m.GetDashboardVersionsCommand{
				Slug: savedDash.Slug,
			}

			err := GetDashboardVersions(&cmd)
			So(err, ShouldBeNil)

			// idk how this actually works
			So(len(cmd.Result), ShouldEqual, 1)
		})

		Convey("Attempt to get the versions for a non-existent slug", func() {
			cmd := m.GetDashboardVersionsCommand{
				Slug: "non-existent-slug",
			}

			err := GetDashboardVersions(&cmd)
			So(err, ShouldNotBeNil)
			So(err, ShouldEqual, m.ErrNoVersionsForSlug)
			So(len(cmd.Result), ShouldEqual, 0)
		})

		Convey("Get all versions for an updated dashboard", func() {
			// Update the dashboard
			updateTestDashboard(savedDash, map[string]interface{}{
				"tags": "different-tag",
			})

			cmd := m.GetDashboardVersionsCommand{
				Slug: savedDash.Slug,
			}
			err := GetDashboardVersions(&cmd)
			So(err, ShouldBeNil)
			So(len(cmd.Result), ShouldEqual, 2)
		})
	})
}

func TestCompareDashboardVersions(t *testing.T) {
	Convey("Testing dashboard version comparison", t, func() {
		InitTestDB(t)
		savedDash := insertTestDashboard("test dash 43", 1, "diff")
		updateTestDashboard(savedDash, map[string]interface{}{
			"tags": "different tag",
		})

		Convey("Compare two versions that are different", func() {
			cmd := m.CompareDashboardVersionsCommand{
				Slug:     savedDash.Slug,
				Original: savedDash.Version,
				New:      savedDash.Version + 1,
			}

			err := CompareDashboardVersionsCommand(&cmd)
			So(err, ShouldBeNil)
			So(cmd.Delta, ShouldNotBeNil)
		})

		Convey("Compare two versions that are the same", func() {
			cmd := m.CompareDashboardVersionsCommand{
				Slug:     savedDash.Slug,
				Original: savedDash.Version,
				New:      savedDash.Version,
			}

			err := CompareDashboardVersionsCommand(&cmd)
			So(err, ShouldBeNil)
			So(cmd.Delta, ShouldBeNil)
		})

		// TODO(ben): diff versions that don't exist to check error condition
	})
}

func TestRestoreDashboardVersion(t *testing.T) {
	Convey("Testing dashboard version restoration", t, func() {
		InitTestDB(t)
		savedDash := insertTestDashboard("test dash 26", 1, "restore")
		updateTestDashboard(savedDash, map[string]interface{}{
			"tags": "not restore",
		})

		Convey("Restore dashboard to a previous version", func() {
			versionsCmd := m.GetDashboardVersionsCommand{
				Slug: savedDash.Slug,
			}
			err := GetDashboardVersions(&versionsCmd)
			So(err, ShouldBeNil)

			cmd := m.RestoreDashboardVersionCommand{
				Slug:    savedDash.Slug,
				Version: savedDash.Version,
			}

			err = RestoreDashboardVersion(&cmd)
			So(err, ShouldBeNil)

			// // Ensure you can restore to each available version
			// for _, version := range versionsCmd.Result {
			// 	cmd := m.RestoreDashboardVersionCommand{
			// 		Slug:    version.Slug,
			// 		Version: version.Version,
			// 	}

			// 	err := RestoreDashboardVersion(&cmd)
			// 	So(err, ShouldBeNil)
			// }
		})

	})
}
