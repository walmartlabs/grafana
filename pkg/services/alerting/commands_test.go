package alerting

import (
	"testing"

	m "github.com/grafana/grafana/pkg/models"
	. "github.com/smartystreets/goconvey/convey"
)

func TestScheduleMissingAlerts(t *testing.T) {
	submitAlertToEngine = func(ruleDef *m.Alert, res []*Rule, factor int) []*Rule {
		res = append(res, &Rule{})
		return res
	}

	Convey("Test scheduling of missing alerts", t, func() {

		Convey("Test No of iterations for alert with frequency of 60s", func() {
			alert := make([]*m.Alert, 0)
			alert1 := &m.Alert{Frequency: 60}
			alert = append(alert, alert1)

			cmd := &ScheduleMissingAlertsCommand{
				MissingAlerts: alert,
			}

			err := scheduleMissingAlerts(cmd)
			So(err, ShouldBeNil)
			So(len(cmd.Result), ShouldEqual, 10)
		})

		Convey("Test No of iterations for alert with frequency of 600s", func() {
			alert := make([]*m.Alert, 0)
			alert1 := &m.Alert{Frequency: 600}
			alert = append(alert, alert1)

			cmd := &ScheduleMissingAlertsCommand{
				MissingAlerts: alert,
			}

			err := scheduleMissingAlerts(cmd)
			So(err, ShouldBeNil)
			So(len(cmd.Result), ShouldEqual, 10)
		})

		Convey("Test No of iterations for alert with frequency of 840s", func() {
			alert := make([]*m.Alert, 0)
			alert1 := &m.Alert{Frequency: 840}
			alert = append(alert, alert1)

			cmd := &ScheduleMissingAlertsCommand{
				MissingAlerts: alert,
			}

			err := scheduleMissingAlerts(cmd)
			So(err, ShouldBeNil)
			So(len(cmd.Result), ShouldEqual, 1)
		})
	})
}
