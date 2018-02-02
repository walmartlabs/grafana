package clustering

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/grafana/grafana/pkg/bus"
	"github.com/grafana/grafana/pkg/log"
	"github.com/grafana/grafana/pkg/metrics"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
	"github.com/grafana/grafana/pkg/setting"
	"golang.org/x/sync/errgroup"
)

type ClusterManager struct {
	clusterNodeMgmt      ClusterNodeMgmt
	ticker               *alerting.Ticker // using the ticker from alerting package for now. Should move the impl outside alerting later
	log                  log.Logger
	alertingState        *AlertingState
	dispatcherTaskQ      chan *DispatcherTask
	dispatcherTaskStatus chan *DispatcherTaskStatus
}

const (
	DISPATCHER_TASK_TYPE_ALERTS_PARTITION = 0
	DISPATCHER_TASK_TYPE_ALERTS_MISSING   = 1
	DISPATCHER_TASK_TYPE_CLEANUP          = 2
)

type DispatcherTaskStatus struct {
	taskType int
	success  bool
	errmsg   string
}
type DispatcherTask struct {
	taskType int
	taskInfo interface{}
}

type DispatcherTaskAlertsMissing struct {
	missingAlerts []*m.Alert
}
type DispatcherTaskAlertsPartition struct {
	partId    int
	nodeCount int
	interval  int64
}

type AlertingState struct {
	status                string
	run_type              string
	lastProcessedInterval int64
}

func NewClusterManager() *ClusterManager {
	cm := &ClusterManager{
		clusterNodeMgmt: getClusterNode(),
		ticker:          alerting.NewTicker(time.Now(), time.Second*0, clock.New()),
		log:             log.New("clustering.clusterManager"),
		alertingState: &AlertingState{
			status:                m.CLN_ALERT_STATUS_OFF,
			lastProcessedInterval: 0,
			run_type:              m.CLN_ALERT_RUN_TYPE_NORMAL,
		},
		dispatcherTaskQ:      make(chan *DispatcherTask, 1),
		dispatcherTaskStatus: make(chan *DispatcherTaskStatus, 1),
	}
	return cm
}

func (cm *ClusterManager) Run(parentCtx context.Context) error {
	cm.log.Info("Initializing cluster manager")
	var reterr error = nil
	taskGroup, ctx := errgroup.WithContext(parentCtx)
	taskGroup.Go(func() error { return cm.clusterMgrTicker(ctx) })
	taskGroup.Go(func() error { return cm.alertRulesDispatcher(ctx) })

	if reterr := taskGroup.Wait(); reterr != nil {
		msg := "Cluster manager stopped"
		cm.log.Info(msg, "reason", reterr)
	}

	cm.log.Info("Cluster manager has terminated")
	return reterr
}

func (cm *ClusterManager) clusterMgrTicker(ctx context.Context) error {
	defer func() {
		if err := recover(); err != nil {
			cm.log.Error("Panic: stopping clusterMgrTicker", "error", err, "stack", log.Stack(1))
		}
	}()
	cm.log.Info("clusterMgrTicker started")
	for {
		select {
		case <-ctx.Done():
			cm.log.Info("clusterMgrTicker Done")
			return ctx.Err()
		case x := <-cm.ticker.C: // ticks every second
			if setting.AlertingEnabled && setting.ExecuteAlerts {
				//execute cleanup scheduler everyday at 12.00.00 AM
				if cm.isTimeForCleanup(x) {
					cm.log.Info("Time to run the cleanup scheduler on one node")
					cm.cleanupScheduler()
					if cm.isTimeToExecuteMissingAlerts(x) {
						cm.scheduleMissingAlerts()
					}
					cm.scheduleNormalAlerts()
				} else if cm.isTimeToExecuteMissingAlerts(x) { //execute missing alerts after every 10 minutes
					cm.log.Debug("Time to run the missing alerts scheduler on one node")
					cm.scheduleMissingAlerts()
					cm.scheduleNormalAlerts()
				} else if x.Second() == 0 { //schedule alert execution at the 0th second of every minute i.e every minute
					cm.log.Debug("Time to run the normal alerts scheduler")
					cm.scheduleNormalAlerts()
				}
				if x.Second() != 0 {
					if cm.alertingState.status != m.CLN_ALERT_STATUS_READY && cm.isAlertExecutionCompleted() {
						//Change status in memory
						cm.changeAlertingStateAndRunType(m.CLN_ALERT_STATUS_READY, m.CLN_ALERT_RUN_TYPE_NORMAL)
						//record status in db
						cm.checkin()
					}
				}
			}
		case taskStatus := <-cm.dispatcherTaskStatus:
			cm.handleDispatcherTaskStatus(taskStatus)
		}
	}
}
func (cm *ClusterManager) isTimeForCleanup(time time.Time) bool {
	currentHour := time.Hour()
	interval := currentHour % setting.ClusteringCleanupPeriod
	if interval == 0 && time.Minute() == 0 && time.Second() == 0 {
		return true
	}
	return false
}

func (cm *ClusterManager) isTimeToExecuteMissingAlerts(time time.Time) bool {
	currentMinute := time.Minute()
	interval := currentMinute % setting.DefaultMissingAlertsSchedularTimeMinutes
	if interval == 0 && time.Second() == 0 {
		return true
	}
	return false
}

func (cm *ClusterManager) checkin() {
	if err := cm.clusterNodeMgmt.CheckIn(cm.alertingState, -1); err != nil {
		cm.log.Error("Failed to checkin", "error", err.Error())
	}
}

//FIXME: Find a better way to check if all the jobs submitted to alert engine are executed.
func (cm *ClusterManager) isAlertExecutionCompleted() bool {
	isAlertExecutionComplete := true
	if cm.alertingState.status == m.CLN_ALERT_STATUS_SCHEDULING ||
		(cm.alertingState.status == m.CLN_ALERT_STATUS_PROCESSING && cm.hasPendingAlertJobs()) {
		isAlertExecutionComplete = false
	}
	return isAlertExecutionComplete
}

func (cm *ClusterManager) handleDispatcherTaskStatus(taskStatus *DispatcherTaskStatus) {
	switch taskStatus.taskType {
	case DISPATCHER_TASK_TYPE_ALERTS_PARTITION:
		if taskStatus.success {
			cm.changeAlertingState(m.CLN_ALERT_STATUS_PROCESSING)
		}
	case DISPATCHER_TASK_TYPE_ALERTS_MISSING:
		if taskStatus.success {
			/*Note that we change the status here to ready because
			we want to execute normal alerts along with missing alerts on this node
			as this node is already taken into consideration for partioning alerts based on
			its ready status in last hearbeat*/
			cm.changeAlertingStateAndRunType(m.CLN_ALERT_STATUS_READY, m.CLN_ALERT_RUN_TYPE_NORMAL)
		}
	case DISPATCHER_TASK_TYPE_CLEANUP:
		if taskStatus.success {
			cm.log.Info("Cleanup task completed")
			cm.changeAlertingStateAndRunType(m.CLN_ALERT_STATUS_READY, m.CLN_ALERT_RUN_TYPE_NORMAL)
		}
	default:
		cm.log.Error("Status received on unsupported task type "+string(taskStatus.taskType),
			"status", taskStatus.success, "error", taskStatus.errmsg)
	}

	if !taskStatus.success {
		cm.log.Error("Failed to dispatch/execute task", "error", taskStatus.errmsg)
		cm.changeAlertingStateAndRunType(m.CLN_ALERT_STATUS_READY, m.CLN_ALERT_RUN_TYPE_NORMAL)
	}
}

func (cm *ClusterManager) cleanupScheduler() bool {
	if cm.alertingState.status != m.CLN_ALERT_STATUS_READY {
		return false
	}
	lastHeartbeat, err := cm.clusterNodeMgmt.GetLastHeartbeat()
	if err != nil {
		cm.log.Error("Failed to get last heartbeat", "error", err)
		return true
	}
	cm.changeAlertingStateAndRunType(m.CLN_ALERT_STATUS_SCHEDULING, m.CLN_ALERT_RUN_TYPE_CLEANUP)
	if err := cm.clusterNodeMgmt.CheckIn(cm.alertingState, 1); err != nil {
		cm.log.Debug("Failed to checkin", "error", err.Error())
		cm.log.Info("Other node is running cleanup job")
		cm.changeAlertingStateAndRunType(m.CLN_ALERT_STATUS_READY, m.CLN_ALERT_RUN_TYPE_NORMAL)
		return true // some other node is doing cleanup
	}
	cm.log.Info("Scheduling cleanup")
	dispatchTask := &DispatcherTask{
		taskType: DISPATCHER_TASK_TYPE_CLEANUP,
		taskInfo: lastHeartbeat,
	}
	cm.dispatcherTaskQ <- dispatchTask
	return true
}

func (cm *ClusterManager) hasPendingAlertJobs() bool {
	jobCountQuery := &alerting.PendingAlertJobCountQuery{}
	err := bus.Dispatch(jobCountQuery)
	if err != nil {
		panic(fmt.Sprintf("Failed to get pending alert job count. Error: %v", err))
	}
	cm.log.Debug("Cluster manager ticker - pending alert jobs", "count", jobCountQuery.ResultCount)
	metrics.M_Clustering_Pending_Alert_Jobs.Update(int64(jobCountQuery.ResultCount))
	return jobCountQuery.ResultCount > 0
}

func (cm *ClusterManager) scheduleMissingAlerts() {
	if cm.alertingState.status != m.CLN_ALERT_STATUS_READY {
		return
	}
	cm.changeAlertingStateAndRunType(m.CLN_ALERT_STATUS_SCHEDULING, m.CLN_ALERT_RUN_TYPE_MISSING)
	err := cm.clusterNodeMgmt.CheckInNodeProcessingMissingAlerts(cm.alertingState)
	if err != nil {
		cm.log.Info("Other node is picked to process missing alerts")
		cm.changeAlertingStateAndRunType(m.CLN_ALERT_STATUS_READY, m.CLN_ALERT_RUN_TYPE_NORMAL)
		return
	}
	nodeID, err := cm.clusterNodeMgmt.GetNodeId()
	cm.log.Info("Scheduling missing alerts", "nodeId", nodeID)
	missingAlerts := cm.clusterNodeMgmt.GetMissingAlerts()
	metrics.M_Clustering_Missing_Alerts_Count.Update(int64(len(missingAlerts)))
	cm.log.Debug(fmt.Sprintf("Count of missing alerts %v", len(missingAlerts)))
	if missingAlerts != nil && len(missingAlerts) > 0 {
		alertDispatchTask1 := &DispatcherTask{
			taskType: DISPATCHER_TASK_TYPE_ALERTS_MISSING,
			taskInfo: &DispatcherTaskAlertsMissing{missingAlerts: missingAlerts},
		}
		cm.dispatcherTaskQ <- alertDispatchTask1
	} else {
		cm.changeAlertingStateAndRunType(m.CLN_ALERT_STATUS_READY, m.CLN_ALERT_RUN_TYPE_NORMAL)
	}
}

func (cm *ClusterManager) scheduleNormalAlerts() {
	cm.log.Info("Scheduling normal alerts")
	if cm.alertingState.status != m.CLN_ALERT_STATUS_READY {
		return
	}
	lastHeartbeat, err := cm.clusterNodeMgmt.GetLastHeartbeat()
	if err != nil {
		cm.log.Error("Failed to get last heartbeat", "error", err)
		return
	}
	node := &m.ActiveNode{
		Heartbeat:    lastHeartbeat,
		AlertStatus:  cm.alertingState.status,
		AlertRunType: cm.alertingState.run_type,
	}
	//get node deatils for last heartbeat which contains partitionId.
	activeNode, err := cm.clusterNodeMgmt.GetNode(node)
	if err != nil {
		cm.log.Warn("Failed to get node for heartbeat "+strconv.FormatInt(lastHeartbeat, 10), "error", err)
		cm.checkin()
		return
	}
	//Get total node count to distribute alerts among nodes
	nodeCount, err := cm.clusterNodeMgmt.GetActiveNodesCount(lastHeartbeat)
	if err != nil {
		cm.log.Error("Failed to get active node count for heartbeat "+string(lastHeartbeat), "error", err)
		return
	}
	metrics.M_Clustering_Active_Nodes.Update(int64(nodeCount))
	cm.log.Debug(fmt.Sprintf("Total active nodes as %v", nodeCount))
	if nodeCount == 0 {
		cm.log.Warn("Found node count 0")
		return
	}
	cm.changeAlertingStateAndRunType(m.CLN_ALERT_STATUS_SCHEDULING, m.CLN_ALERT_RUN_TYPE_NORMAL)
	alertDispatchTask := &DispatcherTask{
		taskType: DISPATCHER_TASK_TYPE_ALERTS_PARTITION,
		taskInfo: &DispatcherTaskAlertsPartition{
			interval:  lastHeartbeat,
			nodeCount: nodeCount,
			partId:    int(activeNode.PartId),
		},
	}
	cm.dispatcherTaskQ <- alertDispatchTask
}

func (cm *ClusterManager) alertRulesDispatcher(ctx context.Context) error {
	defer func() {
		if err := recover(); err != nil {
			cm.log.Error("Panic: stopping alertRulesDispatcher", "error", err, "stack", log.Stack(1))
		}
	}()
	cm.log.Info("alertRulesDispatcher started")
	for {
		select {
		case <-ctx.Done():
			cm.log.Info("alertRulesDispatcher Done")
			return ctx.Err()
		case task := <-cm.dispatcherTaskQ:
			cm.handleDispatcherTask(task)
		}
	}
}

func (cm *ClusterManager) handleDispatcherTask(task *DispatcherTask) {
	var err error = nil
	switch task.taskType {
	case DISPATCHER_TASK_TYPE_ALERTS_PARTITION:
		taskInfo := task.taskInfo.(*DispatcherTaskAlertsPartition)
		scheduleCmd := &alerting.ScheduleAlertsForPartitionCommand{
			Interval:  taskInfo.interval,
			NodeCount: taskInfo.nodeCount,
			PartId:    taskInfo.partId,
		}
		cm.log.Info("Dispatcher - submitted normal alerts batch")
		err = bus.Dispatch(scheduleCmd)
	case DISPATCHER_TASK_TYPE_ALERTS_MISSING:
		taskInfo := task.taskInfo.(*DispatcherTaskAlertsMissing)
		scheduleCmd := &alerting.ScheduleMissingAlertsCommand{
			MissingAlerts: taskInfo.missingAlerts,
		}
		err = bus.Dispatch(scheduleCmd)
		cm.log.Info("Dispatcher - submitted missing alerts batch")
	case DISPATCHER_TASK_TYPE_CLEANUP:
		ts := task.taskInfo.(int64)
		cm.changeAlertingState(m.CLN_ALERT_STATUS_PROCESSING)
		cm.log.Info("Dispatcher - running cleanup job")
		cmd := &m.ClusteringCleanupCommand{LastHeartbeat: ts}
		err = bus.Dispatch(cmd)
	default:
		err = errors.New("Invalid task type " + string(task.taskType))
		cm.log.Error(err.Error())
	}
	if err != nil {
		cm.dispatcherTaskStatus <- &DispatcherTaskStatus{task.taskType, false, err.Error()}
	} else {
		cm.dispatcherTaskStatus <- &DispatcherTaskStatus{task.taskType, true, ""}
	}
}

func (cm *ClusterManager) changeAlertingState(newState string) {
	cm.log.Info("Alerting state: " + cm.alertingState.status + " -> " + newState)
	cm.alertingState.status = newState
}

func (cm *ClusterManager) changeAlertRunType(runType string) {
	cm.log.Debug("Alerting run type: " + runType)
	cm.alertingState.run_type = runType
}

func (cm *ClusterManager) changeAlertingStateAndRunType(newState string, runType string) {
	cm.changeAlertingState(newState)
	cm.changeAlertRunType(runType)
}
