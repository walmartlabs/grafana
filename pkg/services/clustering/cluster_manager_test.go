package clustering

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/grafana/pkg/bus"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
	"github.com/grafana/grafana/pkg/setting"
	. "github.com/smartystreets/goconvey/convey"
)

func TestClusterManager(t *testing.T) {
	Convey("Validate cluster manager for normal alert processing", t, func() {
		setting.NewConfigContext(&setting.CommandLineArgs{
			HomePath: "../../../",
		})
		setting.AlertingEnabled = true
		setting.ExecuteAlerts = true
		setting.ClusteringEnabled = true

		handlers := &mockHandlers{}
		bus.AddHandler("test", handlers.getPendingJobCount)
		bus.AddHandler("test", handlers.scheduleAlertsForPartitionCommand)

		cm := NewClusterManager()

		Convey("Test alerts scheduling", func() {
			handlers.reset()
			cm.clusterNodeMgmt = &mockClusterNodeMgmt{}

			// currently processing; do nothing
			cm.alertingState.status = m.CLN_ALERT_STATUS_PROCESSING
			handlers.pendingJobCount = 1
			cm.scheduleNormalAlerts()
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_PROCESSING)
			So(cm.isAlertExecutionCompleted(), ShouldBeFalse)

			//currently scheduled for processing; do nothing
			cm.alertingState.status = m.CLN_ALERT_STATUS_SCHEDULING
			handlers.pendingJobCount = 0
			cm.scheduleNormalAlerts()
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_SCHEDULING)

			// normal processing; next interval to process; dispatch successful
			cm.alertingState.status = m.CLN_ALERT_STATUS_READY
			handlers.pendingJobCount = 0
			mockCNM := &mockClusterNodeMgmt{
				nodeId:          "testnode:3000",
				activeNodeCount: 1,
				lastHeartbeat:   1493233500,
				activeNode:      &m.ActiveNode{PartId: 0, AlertStatus: m.CLN_ALERT_STATUS_READY},
			}
			cm.clusterNodeMgmt = mockCNM
			cm.scheduleNormalAlerts()
			fmt.Println("status " + cm.alertingState.status)
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_SCHEDULING)
			So(mockCNM.callCountGetLastHeartbeat, ShouldEqual, 1)
			So(mockCNM.callCountGetNode, ShouldEqual, 1)
			So(mockCNM.callCountGetActiveNodesCount, ShouldEqual, 1)
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_SCHEDULING)
			So(len(cm.dispatcherTaskQ), ShouldEqual, 1)

			// dispatch successful
			handlers.scheduleAlertsForPartitionErr = nil
			task := <-cm.dispatcherTaskQ
			cm.handleDispatcherTask(task)
			So(len(cm.dispatcherTaskStatus), ShouldEqual, 1)
			status := <-cm.dispatcherTaskStatus
			So(status.success, ShouldBeTrue)
			So(status.taskType, ShouldEqual, DISPATCHER_TASK_TYPE_ALERTS_PARTITION)
			cm.handleDispatcherTaskStatus(status)
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_PROCESSING)

			//normal processing: next interval to process; dispatch failed
			cm.alertingState.status = m.CLN_ALERT_STATUS_READY
			cm.alertingState.lastProcessedInterval = 1493233440
			handlers.pendingJobCount = 0
			mockCNM = &mockClusterNodeMgmt{
				nodeId:          "testnode:3000",
				activeNodeCount: 1,
				lastHeartbeat:   1493233500,
				activeNode:      &m.ActiveNode{PartId: 0, AlertStatus: m.CLN_ALERT_STATUS_READY},
			}
			cm.clusterNodeMgmt = mockCNM
			cm.scheduleNormalAlerts()
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_SCHEDULING)
			So(mockCNM.callCountGetLastHeartbeat, ShouldEqual, 1)
			So(mockCNM.callCountGetActiveNodesCount, ShouldEqual, 1)
			So(mockCNM.callCountGetNode, ShouldEqual, 1)
			So(len(cm.dispatcherTaskQ), ShouldEqual, 1)

			// dispatch failed
			handlers.scheduleAlertsForPartitionErr = errors.New("some error")
			task = <-cm.dispatcherTaskQ
			cm.handleDispatcherTask(task)
			So(len(cm.dispatcherTaskStatus), ShouldEqual, 1)
			status = <-cm.dispatcherTaskStatus
			So(status.success, ShouldBeFalse)
			So(status.taskType, ShouldEqual, DISPATCHER_TASK_TYPE_ALERTS_PARTITION)
			cm.handleDispatcherTaskStatus(status)
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_READY)
		})
	})
}

func TestClusterManagerForMissingAlerts(t *testing.T) {
	Convey("Validate cluster manager for missing alert processing", t, func() {
		setting.NewConfigContext(&setting.CommandLineArgs{
			HomePath: "../../../",
		})
		setting.AlertingEnabled = true
		setting.ExecuteAlerts = true
		setting.ClusteringEnabled = true

		handlers := &mockHandlers{}
		//bus.AddHandler("test", handlers.SaveNodeProcessingMissingAlertCommand)
		bus.AddHandler("test", handlers.getMissingAlertsQuery)
		bus.AddHandler("test", handlers.ScheduleMissingAlertsCommand)

		cm := NewClusterManager()

		Convey("Test Missing alerts scheduler successful", func() {
			handlers.reset()
			//Missing Alert Processing flow
			alert1 := &m.Alert{
				Name:     "alert1",
				EvalDate: time.Now(),
			}
			missedAlerts := []*m.Alert{alert1}
			mockCNM := &mockClusterNodeMgmt{
				nodeId:          "testnode:3000",
				activeNodeCount: 1,
				lastHeartbeat:   1493233440,
				activeNode:      &m.ActiveNode{PartId: 0, AlertStatus: m.CLN_ALERT_STATUS_READY},
				missingAlerts:   missedAlerts,
			}
			cm.alertingState.status = m.CLN_ALERT_STATUS_READY
			cm.clusterNodeMgmt = mockCNM
			//Schedule Missing ALerts
			cm.scheduleMissingAlerts()
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_SCHEDULING)
			So(cm.alertingState.run_type, ShouldEqual, m.CLN_ALERT_RUN_TYPE_MISSING)
			So(mockCNM.callCountCheckInNodeProcessingMissingAlerts, ShouldEqual, 1)
			So(mockCNM.callCountGetNodeId, ShouldEqual, 1)
			So(mockCNM.callCountGetMissingAlerts, ShouldEqual, 1)
			// missing alerts dispatch successful
			handlers.scheduleMissingAlertsErr = nil
			missingAlertTask := <-cm.dispatcherTaskQ
			cm.handleDispatcherTask(missingAlertTask)
			missingAlertTaskStatus := <-cm.dispatcherTaskStatus
			So(missingAlertTaskStatus.success, ShouldBeTrue)
			So(missingAlertTaskStatus.taskType, ShouldEqual, DISPATCHER_TASK_TYPE_ALERTS_MISSING)
			cm.handleDispatcherTaskStatus(missingAlertTaskStatus)
			//normal alerts should be scheduled after missing alerts
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_SCHEDULING)
			So(cm.alertingState.run_type, ShouldEqual, m.CLN_ALERT_RUN_TYPE_NORMAL)
			//normal alerts dispatch successful
			handlers.scheduleAlertsForPartitionErr = nil
			normalTask := <-cm.dispatcherTaskQ
			cm.handleDispatcherTask(normalTask)
			So(len(cm.dispatcherTaskStatus), ShouldEqual, 1)
			status := <-cm.dispatcherTaskStatus
			cm.handleDispatcherTaskStatus(status)
		})

		Convey("Test Missing alerts are not scheduled as node is not in ready state", func() {
			handlers.reset()
			cm.alertingState.status = m.CLN_ALERT_STATUS_PROCESSING
			nodeNotInReadyState := cm.scheduleMissingAlerts()
			So(nodeNotInReadyState, ShouldBeTrue)
		})

		Convey("Test Missing alerts are not scheduled as other node is processing missing alerts", func() {
			handlers.reset()
			mockCNM := &mockClusterNodeMgmt{
				nodeId: "testnode:3000",
			}
			mockCNM.retError = errors.New("Other node is picked to process missing alerts")
			cm.clusterNodeMgmt = mockCNM
			cm.alertingState.status = m.CLN_ALERT_STATUS_READY
			otherNodeProcessingMissingAlerts := cm.scheduleMissingAlerts()
			So(mockCNM.callCountCheckInNodeProcessingMissingAlerts, ShouldEqual, 1)
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_READY)
			So(cm.alertingState.run_type, ShouldEqual, m.CLN_ALERT_RUN_TYPE_NORMAL)
			So(otherNodeProcessingMissingAlerts, ShouldBeTrue)
		})

		Convey("Test Missing alerts are not dispatched as 0 Missing alerts found", func() {
			handlers.reset()
			mockCNM := &mockClusterNodeMgmt{
				nodeId: "testnode:3000",
			}
			cm.clusterNodeMgmt = mockCNM
			cm.alertingState.status = m.CLN_ALERT_STATUS_READY
			missingAlertsNotfound := cm.scheduleMissingAlerts()
			So(mockCNM.callCountCheckInNodeProcessingMissingAlerts, ShouldEqual, 1)
			So(mockCNM.callCountGetNodeId, ShouldEqual, 1)
			So(mockCNM.callCountGetMissingAlerts, ShouldEqual, 1)
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_READY)
			So(cm.alertingState.run_type, ShouldEqual, m.CLN_ALERT_RUN_TYPE_NORMAL)
			So(missingAlertsNotfound, ShouldBeTrue)
		})
	})
}

func TestClusterManagerForCleanupScheduler(t *testing.T) {
	Convey("Validate cluster manager for Cleanup scheduler", t, func() {
		setting.NewConfigContext(&setting.CommandLineArgs{
			HomePath: "../../../",
		})
		setting.AlertingEnabled = true
		setting.ExecuteAlerts = true
		setting.ClusteringEnabled = true

		handlers := &mockHandlers{}
		bus.AddHandler("test", handlers.ClusteringCleanupCommand)
		cm := NewClusterManager()

		Convey("Test Cleanup scheduler", func() {
			handlers.reset()
			//Clean up scheduler Processing flow
			mockCNM := &mockClusterNodeMgmt{
				nodeId:          "testnode:3000",
				lastHeartbeat:   1493233440,
				activeNodeCount: 1,
				activeNode:      &m.ActiveNode{PartId: 0, AlertStatus: m.CLN_ALERT_STATUS_READY},
			}
			cm.alertingState.status = m.CLN_ALERT_STATUS_READY
			cm.clusterNodeMgmt = mockCNM
			cm.cleanupScheduler()
			So(mockCNM.callCountGetLastHeartbeat, ShouldEqual, 1)
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_SCHEDULING)
			So(cm.alertingState.run_type, ShouldEqual, m.CLN_ALERT_RUN_TYPE_CLEANUP)
			So(mockCNM.callCountCheckIn, ShouldEqual, 1)

			// cleanup dispatch successful
			cleanupTask := <-cm.dispatcherTaskQ
			handlers.cleanupSchedulerErr = nil
			cm.handleDispatcherTask(cleanupTask)
			cleanupTaskStatus := <-cm.dispatcherTaskStatus
			So(cleanupTaskStatus.taskType, ShouldEqual, DISPATCHER_TASK_TYPE_CLEANUP)
			So(cleanupTaskStatus.success, ShouldBeTrue)
			cm.handleDispatcherTaskStatus(cleanupTaskStatus)
			//normal alerts should be scheduled after cleanup
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_SCHEDULING)
			So(cm.alertingState.run_type, ShouldEqual, m.CLN_ALERT_RUN_TYPE_NORMAL)

			//clean up not scheduled if node not in ready state
			notInReadyState := cm.cleanupScheduler()
			So(notInReadyState, ShouldBeTrue)
		})

		Convey("Test Cleanup not scheduled if other node is processing cleanup", func() {
			handlers.reset()
			mockCNM := &mockClusterNodeMgmt{
				nodeId:        "testnode:3000",
				lastHeartbeat: 1493233441,
			}
			mockCNM.checkInError = errors.New("Failed to checkin. Other node is running cleanup job")
			cm.alertingState.status = m.CLN_ALERT_STATUS_READY
			cm.clusterNodeMgmt = mockCNM
			isOtherNodeDoingCleanup := cm.cleanupScheduler()
			So(mockCNM.callCountGetLastHeartbeat, ShouldEqual, 1)
			So(mockCNM.callCountCheckIn, ShouldEqual, 1)
			So(cm.alertingState.status, ShouldEqual, m.CLN_ALERT_STATUS_READY)
			So(cm.alertingState.run_type, ShouldEqual, m.CLN_ALERT_RUN_TYPE_NORMAL)
			So(isOtherNodeDoingCleanup, ShouldBeTrue)
		})
	})
}

type mockHandlers struct {
	pendingJobCount               int
	alerts                        []*m.Alert
	scheduleAlertsForPartitionErr error
	scheduleMissingAlertsErr      error
	cleanupSchedulerErr           error
}

func (mh *mockHandlers) reset() {
	mh.pendingJobCount = 0
	mh.alerts = nil
	mh.scheduleAlertsForPartitionErr = nil
	mh.scheduleMissingAlertsErr = nil
}
func (mh *mockHandlers) getPendingJobCount(query *alerting.PendingAlertJobCountQuery) error {
	query.ResultCount = mh.pendingJobCount
	return nil
}
func (mh *mockHandlers) getMissingAlertsQuery(query *m.GetMissingAlertsQuery) error {
	query.Result = mh.alerts
	return nil
}
func (mh *mockHandlers) scheduleAlertsForPartitionCommand(cmd *alerting.ScheduleAlertsForPartitionCommand) error {
	return mh.scheduleAlertsForPartitionErr
}

func (mh *mockHandlers) ScheduleMissingAlertsCommand(cmd *alerting.ScheduleMissingAlertsCommand) error {
	return mh.scheduleMissingAlertsErr
}

func (mh *mockHandlers) ClusteringCleanupCommand(cmd *m.ClusteringCleanupCommand) error {
	return mh.cleanupSchedulerErr
}

type mockClusterNodeMgmt struct {
	retError                                    error
	nodeId                                      string
	activeNode                                  *m.ActiveNode
	activeNodeCount                             int
	lastHeartbeat                               int64
	missingAlerts                               []*m.Alert
	nodeProcessingMissingAlert                  *m.ActiveNode
	callCountGetNodeId                          int
	callCountCheckIn                            int
	callCountGetNode                            int
	callCountCheckInNodeProcessingMissingAlerts int
	callCountGetActiveNodesCount                int
	callCountGetLastHeartbeat                   int
	callCountGetMissingAlerts                   int
	callCountGetNodeProcessingMissingAlerts     int
	checkInError                                error
}

func (cn *mockClusterNodeMgmt) GetNodeId() (string, error) {
	cn.callCountGetNodeId++
	return cn.nodeId, cn.retError
}
func (cn *mockClusterNodeMgmt) CheckIn(alertingState *AlertingState, participantLimit int) error {
	cn.callCountCheckIn++
	return cn.checkInError
}
func (cn *mockClusterNodeMgmt) GetNode(node *m.ActiveNode) (*m.ActiveNode, error) {
	cn.callCountGetNode++
	return cn.activeNode, cn.retError
}
func (cn *mockClusterNodeMgmt) CheckInNodeProcessingMissingAlerts(alertingState *AlertingState) error {
	cn.callCountCheckInNodeProcessingMissingAlerts++
	return cn.retError
}
func (cn *mockClusterNodeMgmt) GetActiveNodesCount(heartbeat int64) (int, error) {
	cn.callCountGetActiveNodesCount++
	return cn.activeNodeCount, cn.retError
}
func (cn *mockClusterNodeMgmt) GetLastHeartbeat() (int64, error) {
	cn.callCountGetLastHeartbeat++
	return cn.lastHeartbeat, cn.retError
}

func (cn *mockClusterNodeMgmt) GetMissingAlerts() []*m.Alert {
	cn.callCountGetMissingAlerts++
	return cn.missingAlerts
}

func (cn *mockClusterNodeMgmt) GetNodeProcessingMissingAlerts() *m.ActiveNode {
	cn.callCountGetNodeProcessingMissingAlerts++
	return cn.nodeProcessingMissingAlert
}
