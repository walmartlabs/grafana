package alerting

import (
	"errors"
	"fmt"
	"time"

	"github.com/grafana/grafana/pkg/bus"
	m "github.com/grafana/grafana/pkg/models"
	s "github.com/grafana/grafana/pkg/setting"
)

type UpdateDashboardAlertsCommand struct {
	UserId    int64
	OrgId     int64
	Dashboard *m.Dashboard
}

type ValidateDashboardAlertsCommand struct {
	UserId    int64
	OrgId     int64
	Dashboard *m.Dashboard
}

type PendingAlertJobCountQuery struct {
	ResultCount int
}

type ScheduleAlertsForPartitionCommand struct {
	PartId    int
	NodeCount int
	Interval  int64
}

type ScheduleMissingAlertsCommand struct {
	MissingAlerts []*m.Alert
	Result        []*Rule
}

func init() {
	bus.AddHandler("alerting", updateDashboardAlerts)
	bus.AddHandler("alerting", validateDashboardAlerts)
	bus.AddHandler("alerting", getPendingAlertJobCount)
	bus.AddHandler("alerting", scheduleAlertsForPartition)
	bus.AddHandler("alerting", scheduleMissingAlerts)
}

func validateDashboardAlerts(cmd *ValidateDashboardAlertsCommand) error {
	extractor := NewDashAlertExtractor(cmd.Dashboard, cmd.OrgId)

	if _, err := extractor.GetAlerts(); err != nil {
		return err
	}

	return nil
}

func updateDashboardAlerts(cmd *UpdateDashboardAlertsCommand) error {
	saveAlerts := m.SaveAlertsCommand{
		OrgId:       cmd.OrgId,
		UserId:      cmd.UserId,
		DashboardId: cmd.Dashboard.Id,
	}

	extractor := NewDashAlertExtractor(cmd.Dashboard, cmd.OrgId)

	if alerts, err := extractor.GetAlerts(); err != nil {
		return err
	} else {
		saveAlerts.Alerts = alerts
	}

	if err := bus.Dispatch(&saveAlerts); err != nil {
		return err
	}

	return nil
}

func getPendingAlertJobCount(query *PendingAlertJobCountQuery) error {
	if engine == nil {
		return errors.New("Alerting engine is not initialized")
	}
	query.ResultCount = len(engine.execQueue)
	return nil
}

func scheduleAlertsForPartition(cmd *ScheduleAlertsForPartitionCommand) error {
	if engine == nil {
		return errors.New("Alerting engine is not initialized")
	}
	if cmd.NodeCount == 0 {
		return errors.New("Node count is 0")
	}
	if cmd.PartId >= cmd.NodeCount {
		return errors.New(fmt.Sprintf("Invalid partition id %v (node count = %v)", cmd.PartId, cmd.NodeCount))
	}
	rules := engine.ruleReader.Fetch()
	filterCount := 0
	intervalEnd := time.Unix(cmd.Interval, 0).Add(time.Minute)
	for _, rule := range rules {
		evalDateTrunc := rule.EvalDate.Truncate(time.Minute)
		// handle frequency greater than 1 min
		nextEvalDate := evalDateTrunc.Add(time.Duration(rule.Frequency) * time.Second)
		if nextEvalDate.Before(intervalEnd) || nextEvalDate.Equal(intervalEnd) {
			if rule.Id%int64(cmd.NodeCount) == int64(cmd.PartId) {
				engine.execQueue <- &Job{Rule: rule}
				filterCount++
				engine.log.Debug(fmt.Sprintf("Scheduled Rule : %v for interval=%v", rule, cmd.Interval))
			} else {
				engine.log.Debug(fmt.Sprintf("Skipped Rule : %v for interval=%v, partition id=%v, nodeCount=%v", rule, cmd.Interval, cmd.PartId, cmd.NodeCount))
			}
		} else {
			engine.log.Debug(fmt.Sprintf("Skipped Rule : %v for interval=%v, intervalEnd=%v, nextEvalDate=%v", rule, cmd.Interval, intervalEnd, nextEvalDate))
		}
	}
	engine.log.Info(fmt.Sprintf("%v/%v rules scheduled for execution for partition %v/%v",
		filterCount, len(rules), cmd.PartId, cmd.NodeCount))
	return nil
}

func scheduleMissingAlerts(cmd *ScheduleMissingAlertsCommand) error {
	//transform each alert to rule
	res := make([]*Rule, 0)
	missingAlerts := cmd.MissingAlerts
	for _, ruleDef := range missingAlerts {
		frequency := ruleDef.Frequency
		/*
			If frequency is in multiples of 60 sec and less than 10 min = (600 seconds).
			*Then we evaluate datapoints for all the values in past 10 minutes because we introduced a delay of
			*10 minutes when getting the missing alerts.Check sqlstore/services/alert.go
			*Note that we don't evaluate datapoint for alerts with frequency less than 60 seconds because
			*the frequency is too frequent and won't make sense to excute missing alerts on low frequency.
		*/
		noOfIterations := int(s.DefaultMissingAlertsDelay / 60) //10 iterations
		if frequency >= 60 || frequency <= s.DefaultMissingAlertsDelay {
			factor := 1
			for factor <= noOfIterations {
				submitAlertToEngine(ruleDef, res, factor)
				factor += factor
			}
		} else if frequency > s.DefaultMissingAlertsDelay { //For frequency greater than 10 minutes just go back to previous missed frequency
			frequencyInMin := int(frequency / 60)
			factor := 1 + frequencyInMin
			submitAlertToEngine(ruleDef, res, factor)
		}
	}
	cmd.Result = res
	engine.log.Info(fmt.Sprintf("Total no of rules scheduled for execution of missed alerts is %v", len(missingAlerts)))
	return nil
}

func submitAlertToEngine(ruleDef *m.Alert, res []*Rule, factor int) {
	if model, err := ModifiedRuleFromDBAlert(ruleDef, factor); err != nil {
		engine.log.Error("Could not build alert model for rule", "ruleId", ruleDef.Id, "error", err)
	} else {
		res = append(res, model)
		engine.execQueue <- &Job{Rule: model}
		engine.log.Debug(fmt.Sprintf("Scheduled missed Rule : %v", model.Name))
	}
}
