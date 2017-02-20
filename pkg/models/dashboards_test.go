package models

import (
	"testing"

	"github.com/grafana/grafana/pkg/components/simplejson"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDashboardModel(t *testing.T) {

	Convey("When generating slug", t, func() {
		dashboard := NewDashboard("Grafana Play Home")
		dashboard.UpdateSlug()

		So(dashboard.Slug, ShouldEqual, "grafana-play-home")
	})

	Convey("Given a dashboard json", t, func() {
		json := simplejson.New()
		json.Set("title", "test dash")

		Convey("With tags as string value", func() {
			json.Set("tags", "")
			dash := NewDashboardFromJson(json)

			So(len(dash.GetTags()), ShouldEqual, 0)
		})
	})
}

func TestDashboardDiff(t *testing.T) {
	d := NewDashboard("diff")

	a := simplejson.New()
	a.Set("key", "val")

	b := simplejson.New()
	b.Set("key", "val2")

	diff, err := d.Diff(a, b)

	if err != nil {
		t.Fatalf("Expected diff to succeed but got %v\n", err)
	}

	// Check to make sure it's not nil
	if diff == nil {
		t.Fatal("Expected diff to be not nil\n")
	}

	// Output the diff to check visually
	println(string(diff))
}
