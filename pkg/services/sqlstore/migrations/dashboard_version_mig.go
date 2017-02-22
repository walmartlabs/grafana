package migrations

import . "github.com/grafana/grafana/pkg/services/sqlstore/migrator"

func addDashboardVersionMigration(mg *Migrator) {
	dashboardVersionV1 := Table{
		Name: "dashboard_version",
		Columns: []*Column{
			{Name: "id", Type: DB_BigInt, IsPrimaryKey: true, IsAutoIncrement: true},
			{Name: "dashboard_id", Type: DB_BigInt},
			{Name: "slug", Type: DB_NVarchar, Length: 255, Nullable: false},
			{Name: "version", Type: DB_Int, Nullable: false},
			{Name: "created", Type: DB_DateTime, Nullable: false},
			{Name: "created_by", Type: DB_BigInt, Nullable: false},
			{Name: "message", Type: DB_Text, Nullable: false},
			{Name: "data", Type: DB_Text, Nullable: false},
		},
	}

	mg.AddMigration("create dashboard_version table v1", NewAddTableMigration(dashboardVersionV1))
}
