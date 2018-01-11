package alerting

import (
	"testing"

	"github.com/grafana/grafana/pkg/components/simplejson"
	m "github.com/grafana/grafana/pkg/models"
	. "github.com/smartystreets/goconvey/convey"
)

type FakeCondition struct{}

func (f *FakeCondition) Eval(context *EvalContext) (*ConditionResult, error) {
	return &ConditionResult{}, nil
}

func TestAlertRuleModel(t *testing.T) {
	Convey("Testing alert rule", t, func() {

		RegisterCondition("test", func(model *simplejson.Json, index int) (Condition, error) {
			return &FakeCondition{}, nil
		})

		Convey("Can parse seconds", func() {
			seconds, _ := getTimeDurationStringToSeconds("10s")
			So(seconds, ShouldEqual, 10)
		})

		Convey("Can parse minutes", func() {
			seconds, _ := getTimeDurationStringToSeconds("10m")
			So(seconds, ShouldEqual, 600)
		})

		Convey("Can parse hours", func() {
			seconds, _ := getTimeDurationStringToSeconds("1h")
			So(seconds, ShouldEqual, 3600)
		})

		Convey("defaults to seconds", func() {
			seconds, _ := getTimeDurationStringToSeconds("1o")
			So(seconds, ShouldEqual, 1)
		})

		Convey("should return err for empty string", func() {
			_, err := getTimeDurationStringToSeconds("")
			So(err, ShouldNotBeNil)
		})

		Convey("can construct alert rule model", func() {
			json := `
			{
				"name": "name2",
				"description": "desc2",
				"handler": 0,
				"noDataMode": "critical",
				"enabled": true,
				"frequency": "60s",
        "conditions": [
          {
            "type": "test",
            "prop": 123
					}
        ],
        "notifications": [
					{"id": 1134},
					{"id": 22}
				]
			}
			`

			alertJSON, jsonErr := simplejson.NewJson([]byte(json))
			So(jsonErr, ShouldBeNil)

			alert := &m.Alert{
				Id:          1,
				OrgId:       1,
				DashboardId: 1,
				PanelId:     1,

				Settings: alertJSON,
			}

			alertRule, err := NewRuleFromDBAlert(alert)
			So(err, ShouldBeNil)

			So(len(alertRule.Conditions), ShouldEqual, 1)

			Convey("Can read notifications", func() {
				So(len(alertRule.Notifications), ShouldEqual, 2)
			})
		})
	})
}

func TestModifiedAlertRuleModel(t *testing.T) {
	Convey("Testing modified alert rule", t, func() {

		RegisterCondition("query", func(model *simplejson.Json, index int) (Condition, error) {
			return &FakeCondition{}, nil
		})

		Convey("can construct modified alert rule model", func() {
			json := `
			{
				"name": "name2",
				"description": "desc2",
				"handler": 0,
				"noDataMode": "critical",
				"enabled": true,
				"frequency": "60s",
				"conditions": [
					{
					  "query": {
						"params": [
						  "B",
						  "3m",
						  "now"
						]
					  },
					  "type": "query"
					},
					{
					  "query": {
						"params": [
						  "D",
						  "5m",
						  "now"
						]
					  },
					  "type": "query"
					}
				  ],
				  "notifications": [
					{"id": 1134},
					{"id": 22}
				]
			}
			`
			alertJSON, jsonErr := simplejson.NewJson([]byte(json))
			So(jsonErr, ShouldBeNil)

			alert := &m.Alert{
				Id:          1,
				OrgId:       1,
				DashboardId: 1,
				PanelId:     1,

				Settings: alertJSON,
			}

			modifiedAlertRule, err := ModifiedRuleFromDBAlert(alert, 2)
			So(err, ShouldBeNil)
			So(len(modifiedAlertRule.Conditions), ShouldEqual, 2)
			for _, condition := range alert.Settings.Get("conditions").MustArray() {
				conditionModel := simplejson.NewFromAny(condition)
				queryMap := conditionModel.Get("query").MustMap()
				params, _ := queryMap["params"]
				paramsArray := params.([]interface{})
				So(paramsArray[2], ShouldEqual, "now-2m")
			}
		})
	})

}
