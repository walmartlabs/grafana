package sqlstore

import (
	"testing"

	m "github.com/grafana/grafana/pkg/models"
	. "github.com/smartystreets/goconvey/convey"
)

func TestActiveNode(t *testing.T) {
	Convey("Testing insert active node heartbeat", t, func() {
		InitTestDB(t)
		act := m.ActiveNode{
			NodeId:       "10.0.0.1:3030",
			AlertRunType: m.CLN_ALERT_RUN_TYPE_NORMAL,
			AlertStatus:  m.CLN_ALERT_STATUS_READY,
		}
		cmd1 := m.SaveActiveNodeCommand{
			Node:        &act,
			FetchResult: true,
		}

		err := InsertActiveNodeHeartbeat(&cmd1)
		Convey("Can  insert active node", func() {
			So(err, ShouldBeNil)
		})

		Convey("Retrive data", func() {
			So(cmd1.Result, ShouldNotBeNil)
			So(cmd1.Result.NodeId, ShouldEqual, "10.0.0.1:3030")
			So(cmd1.Result.Heartbeat, ShouldBeGreaterThan, 0)
			So(cmd1.Result.PartId, ShouldEqual, 0)
		})

		/*
		*Test insertion of node processing missing alerts
		 */
		nodeID := "10.1.1.1:4330"
		cmd2 := m.SaveNodeProcessingMissingAlertCommand{
			Node: &m.ActiveNode{
				NodeId:       nodeID,
				AlertRunType: m.CLN_ALERT_RUN_TYPE_MISSING,
				AlertStatus:  m.CLN_ALERT_STATUS_SCHEDULING,
			},
		}
		err2 := InsertNodeProcessingMissingAlert(&cmd2)
		Convey("Can  insert node processing missing alert", func() {
			So(err2, ShouldBeNil)
		})
		cmd3 := m.GetNodeProcessingMissingAlertsCommand{}
		err3 := GetNodeProcessingMissingAlerts(&cmd3)
		Convey("Retrive Node Processing Missing Alert", func() {
			So(err3, ShouldBeNil)
			So(cmd3.Result.NodeId, ShouldEqual, nodeID)
			So(cmd3.Result.Heartbeat, ShouldBeGreaterThan, 0)
			So(cmd3.Result.PartId, ShouldEqual, 0)
			So(cmd3.Result.AlertRunType, ShouldEqual, m.CLN_ALERT_RUN_TYPE_MISSING)
			So(cmd3.Result.AlertStatus, ShouldEqual, m.CLN_ALERT_STATUS_SCHEDULING)
		})

		// Get Last heartbeat
		cmd4 := m.GetLastDBTimeIntervalQuery{}
		err4 := GetLastDBTimeInterval(&cmd4)
		Convey("Can  get last heartbeat", func() {
			So(err4, ShouldBeNil)
		})
		lastHeartbeat := cmd4.Result
		Convey("getting last heartbeat", func() {
			So(lastHeartbeat, ShouldNotBeNil)
		})

		//Get active nodes count for last heartbeat
		hb := cmd1.Result.Heartbeat
		cmd5 := m.GetActiveNodesCountCommand{
			Heartbeat: hb,
		}
		err = GetActiveNodesCount(&cmd5)
		Convey("Can  get active node count", func() {
			So(err, ShouldBeNil)
		})
		countOfaciveNodes := cmd5.Result
		Convey("getting active node count", func() {
			So(countOfaciveNodes, ShouldEqual, 1)
		})

		//Get Node based on heartbeat,node_id,alert_status and alert_run_type
		act.Heartbeat = hb
		cmd6 := m.GetNodeCmd{
			Node: &act,
		}
		err = GeNode(&cmd6)
		Convey("Get Node", func() {
			So(cmd6.Result, ShouldNotBeNil)
			So(cmd6.Result.NodeId, ShouldEqual, "10.0.0.1:3030")
			So(cmd6.Result.Heartbeat, ShouldEqual, hb)
			So(cmd6.Result.PartId, ShouldEqual, 0)
			So(cmd6.Result.AlertRunType, ShouldEqual, m.CLN_ALERT_RUN_TYPE_NORMAL)
			So(cmd6.Result.AlertStatus, ShouldEqual, m.CLN_ALERT_STATUS_READY)
		})
	})
}
