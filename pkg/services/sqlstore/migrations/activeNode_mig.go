package migrations

import (
	. "github.com/grafana/grafana/pkg/services/sqlstore/migrator"
)

func addActiveNodeMigration(mg *Migrator) {
	active_node := Table{
		Name: "active_node",
		Columns: []*Column{
			{Name: "id", Type: DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "node_id", Type: DB_NVarchar, Length: 255, Nullable: false},
			{Name: "heartbeat", Type: DB_BigInt, Nullable: false},
			{Name: "partitionNo", Type: DB_Int, Nullable: false},
			{Name: "alertRunType", Type: DB_Varchar, Length: 50, Nullable: false},
		},
		Indices: []*Index{
			{Cols: []string{"node_id", "heartbeat"}, Type: UniqueIndex},
			{Cols: []string{"heartbeat", "partitionNo", "alertRunType"}, Type: UniqueIndex},
		},
	}
	mg.AddMigration("create active_node table", NewAddTableMigration(active_node))
	mg.AddMigration("add index active_node.node_id_heartbeat", NewAddIndexMigration(active_node, active_node.Indices[0]))
	mg.AddMigration("add unique index active_node.partitionNo_heartbeat_alertRunType", NewAddIndexMigration(active_node, active_node.Indices[1]))
}
