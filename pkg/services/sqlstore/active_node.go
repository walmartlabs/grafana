package sqlstore

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/grafana/grafana/pkg/bus"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/setting"
)

var (
	insertHeartbeatSQL  = "insert into active_node(node_id, heartbeat, part_id, alert_run_type, alert_status) values(?, ?, ?, ?, ?)"
	getNextPartIDSQL    = "select count(part_id) as part_id from active_node where heartbeat = ? and alert_status = ?"
	lastCleanupCheckSQL = "select * from active_node as a where a.heartbeat > ? and a.heartbeat <= ? and alert_run_type='" + m.CLN_ALERT_RUN_TYPE_CLEANUP + "'"
	deleteHearbeatSQL   = "delete from active_node where heartbeat < ?"
	deleteAnnotationSQL = "delete from annotation where epoch < ?"
)

func init() {
	bus.AddHandler("sql", GetActiveNodeByIdHeartbeat)
	bus.AddHandler("sql", InsertActiveNodeHeartbeat)
	bus.AddHandler("sql", InsertNodeProcessingMissingAlert)
	bus.AddHandler("sql", GetLastDBTimeInterval)
	bus.AddHandler("sql", GetActiveNodesCount)
	bus.AddHandler("sql", GetNodeProcessingMissingAlerts)
	bus.AddHandler("sql", ClusteringCleanup)
}

func GetActiveNodeByIdHeartbeat(query *m.GetActiveNodeByIdHeartbeatQuery) error {
	var retNode m.ActiveNode
	has, err := x.Where("heartbeat=?", query.Heartbeat).And("node_id=?", query.NodeId).Get(&retNode)
	if err != nil || !has {
		errmsg := fmt.Sprintf("Heartbeat record not found: nodeId=%s, heartbeat=%d", query.NodeId, query.Heartbeat)
		if err == nil {
			err = errors.New(errmsg)
			sqlog.Debug(errmsg)
		} else {
			sqlog.Error(errmsg, "error", err)
		}
		return err
	}
	query.Result = &retNode
	return nil
}

func InsertActiveNodeHeartbeat(cmd *m.SaveActiveNodeCommand) error {
	sqlog.Debug(fmt.Sprintf("Received command %v", cmd))
	if cmd.Node == nil {
		return errors.New("No ActiveNode found to save")
	}
	if !validAlertRunType(cmd.Node.AlertRunType) {
		errmsg := "Invalid alert run type " + cmd.Node.AlertRunType
		sqlog.Error(errmsg)
		return errors.New(errmsg)
	}
	if !validAlertStatus(cmd.Node.AlertStatus) {
		errmsg := "Invalid alert status " + cmd.Node.AlertStatus
		sqlog.Error(errmsg)
		return errors.New(errmsg)
	}
	var ts int64 = -1
	retryCount := 3
	var err error
	for retryCount > 0 {
		err = inTransaction(func(sess *DBSession) error {

			results, err := sess.Query("select " + dialect.CurrentTimeToRoundMinSql() + " as ts ")
			if err != nil {
				errmsg := "Failed to get timestamp"
				sqlog.Debug(errmsg, "error", err)
				return errors.New(errmsg + ": " + err.Error())
			}
			ts, err = strconv.ParseInt(string(results[0]["ts"]), 10, 64)
			if err != nil {
				errmsg := "Failed to get timestamp"
				sqlog.Debug(errmsg, "error", err)
				return errors.New(errmsg + ": " + err.Error())
			}
			results, err = sess.Query(getNextPartIDSQL, ts, cmd.Node.AlertStatus)
			if err != nil {
				errmsg := "Failed to get next part_id"
				sqlog.Debug(errmsg, "error", err)
				return errors.New(errmsg + ": " + err.Error())
			}
			partID, err := strconv.ParseInt(string(results[0]["part_id"]), 10, 64)
			if err != nil {
				errmsg := "Failed to get next part_id"
				sqlog.Debug(errmsg, "error", err)
				return errors.New(errmsg + ": " + err.Error())
			}
			if cmd.ParticipantLimit > 0 && partID == int64(cmd.ParticipantLimit) {
				retryCount = 0
				return errors.New("Participant limit reached")
			}
			_, err = sess.Exec(insertHeartbeatSQL, cmd.Node.NodeId, ts, partID, cmd.Node.AlertRunType, cmd.Node.AlertStatus)
			if err != nil {
				errmsg := "Failed to insert heartbeat"
				sqlog.Debug(errmsg, "error", err)
				return errors.New(errmsg + ": " + err.Error())
			}
			sqlog.Debug("Active node heartbeat inserted", "id", cmd.Node.Id)
			return nil
		})
		if err != nil {
			sqlog.Debug("Insert heartbeat tx failed. Retrying...", "error", err)
		} else {
			break
		}
		retryCount--
	}

	if err != nil {
		sqlog.Error("Insert heartbeat tx failed", "error", err)
		return err
	}
	if cmd.FetchResult {
		getcmd := m.GetActiveNodeByIdHeartbeatQuery{NodeId: cmd.Node.NodeId, Heartbeat: ts}
		err = GetActiveNodeByIdHeartbeat(&getcmd)
		if err != nil {
			return err
		}
		cmd.Result = getcmd.Result
	}
	return nil
}

func InsertNodeProcessingMissingAlert(cmd *m.SaveNodeProcessingMissingAlertCommand) error {
	return inTransaction(func(sess *DBSession) error {
		results, err := sess.Query("select " + dialect.CurrentTimeToRoundMinSql() + " as ts ")
		if err != nil {
			sqlog.Error("Failed to get timestamp", "error", err)
			return err
		}
		ts, err := strconv.ParseInt(string(results[0]["ts"]), 10, 64)
		if err != nil {
			sqlog.Error("Failed to get timestamp", "error", err)
			return err
		}
		nodeProcessingMissingAlert := &m.ActiveNode{
			NodeId:       cmd.Node.NodeId,
			PartId:       0,
			AlertRunType: cmd.Node.AlertRunType,
			Heartbeat:    ts,
			AlertStatus:  cmd.Node.AlertStatus,
		}
		if _, err = sess.Insert(nodeProcessingMissingAlert); err != nil {
			return err
		}
		cmd.Result = nodeProcessingMissingAlert
		return nil
	})
}

func GetLastDBTimeInterval(cmd *m.GetLastDBTimeIntervalQuery) error {
	if cmd == nil {
		return errors.New("Invalid command received to GetLastDBTimeInterval")
	}
	results, err := x.Query("select " + dialect.CurrentTimeToRoundMinSql() + " as ts ")
	if err != nil {
		sqlog.Error("Failed to get db timestamp", "error", err)
		return err
	}
	ts, err := strconv.ParseInt(string(results[0]["ts"]), 10, 64)
	if err != nil {
		sqlog.Error("Failed to get db timestamp", "error", err)
		return err
	}
	cmd.Result = ts - 60
	return nil
}

func validAlertRunType(status string) bool {
	switch status {
	case m.CLN_ALERT_RUN_TYPE_MISSING:
	case m.CLN_ALERT_RUN_TYPE_NORMAL:
	case m.CLN_ALERT_RUN_TYPE_CLEANUP:
	default:
		return false
	}
	return true
}

func validAlertStatus(status string) bool {
	switch status {
	case m.CLN_ALERT_STATUS_OFF:
	case m.CLN_ALERT_STATUS_READY:
	case m.CLN_ALERT_STATUS_PROCESSING:
	case m.CLN_ALERT_STATUS_SCHEDULING:
	default:
		return false
	}
	return true
}

func GetActiveNodesCount(cmd *m.GetActiveNodesCountCommand) error {
	var actNodes []m.ActiveNode
	err := x.Where("heartbeat=?", cmd.Heartbeat).And("alert_status=?", m.CLN_ALERT_STATUS_READY).Find(&actNodes)
	if err != nil || (len(actNodes) == 0) {
		errmsg := fmt.Sprintf("Failed to get node count for heartbeat=%d", cmd.Heartbeat)
		if err == nil {
			err = errors.New(errmsg)
			sqlog.Error(errmsg)
		} else {
			sqlog.Error(errmsg, "error", err)
		}
		return err
	}
	cmd.Result = len(actNodes)
	return nil
}

func GetNodeProcessingMissingAlerts(cmd *m.GetNodeProcessingMissingAlertsCommand) error {
	var retNode m.ActiveNode
	results, err1 := x.Query("select " + dialect.CurrentTimeToRoundMinSql() + " as ts ")
	if err1 != nil {
		sqlog.Error("Failed to get db timestamp", "error", err1)
		return err1
	}
	ts, _ := strconv.ParseInt(string(results[0]["ts"]), 10, 64)
	_, err := x.Where("heartbeat=?", ts).And("alert_run_type=?", m.CLN_ALERT_RUN_TYPE_MISSING).Get(&retNode)
	if err != nil {
		errmsg := fmt.Sprintf("Failed to get Node processing missing alert for heartbeat=%d", ts)
		sqlog.Error(errmsg, "error", err)
		return err
	}
	cmd.Result = &retNode
	return nil
}

// func ClusteringCleanupCheck(cmd *m.ClusteringCleanupCheckCommand) error {
// 	sqlog.Debug("ClusteringCleanupCheck called")
// 	lasthb := cmd.LastHeartbeat
// 	results, err1 := x.Query(lastCleanupCheckSQL, lasthb-int64(setting.ClusteringCleanupPeriod), lasthb)
// 	if err1 != nil {
// 		sqlog.Warn("Cleanup check failed", "error", err1)
// 		cmd.Result = false
// 		return err1
// 	}
// 	if len(results) > 0 {
// 		sqlog.Debug("Cheanup is already done")
// 		cmd.Result = false
// 		return nil
// 	}
// 	cmd.Result = true
// 	return nil
// }

func ClusteringCleanup(cmd *m.ClusteringCleanupCommand) error {
	sqlog.Debug("ClusteringCleanup called")
	lasthb := cmd.LastHeartbeat
	var reterr error
	txerr := inTransaction(func(sess *DBSession) error {
		result, err := sess.Exec(deleteHearbeatSQL, lasthb-int64(setting.ClusteringHBRetention))
		if err != nil {
			sqlog.Error("Heartbeat cleanup failed", "error", err)
			return err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			sqlog.Error("Heartbeat cleanup failed", "error", err)
			return err
		}
		sqlog.Info("'active_node' table cleanup done", "rows deleted", rowsAffected)
		return nil
	})
	if txerr != nil {
		reterr = txerr
	}
	txerr = inTransaction(func(sess *DBSession) error {
		annoresult, err := sess.Exec(deleteAnnotationSQL, lasthb-int64(setting.AnnotationRetention))
		if err != nil {
			sqlog.Error("Annotation cleanup failed", "error", err)
			return err
		}
		rowsAffected, err := annoresult.RowsAffected()
		if err != nil {
			sqlog.Error("Annotation cleanup failed", "error", err)
			return err
		}
		sqlog.Info("'annotation' table cleanup done", "rows deleted", rowsAffected)
		return nil
	})
	if reterr == nil && txerr != nil {
		reterr = txerr
	}
	return reterr

}
