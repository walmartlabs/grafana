///<reference path="../../../headers/common.d.ts" />

import './audit_ctrl';

import _ from 'lodash';
import coreModule from 'app/core/core_module';
import {DashboardModel} from '../model';

export class AuditSrv {
  /** @ngInject */
  constructor(private backendSrv, private $q) {}

  getAuditLog(dashboard: DashboardModel) {
    const id = dashboard && dashboard.id ? dashboard.id : void 0;
    return id ? this.backendSrv.get(`api/dashboards/db/${id}/versions`) : this.$q.when([]);
  }

  compareVersions(dashboard: DashboardModel, selected: [number, number]) {
    const id = dashboard && dashboard.id ? dashboard.id : void 0;
    const url = `api/dashboards/db/${id}/compare/${_.min(selected)}...${_.max(selected)}/basic`;
    return id ? this.backendSrv.get(url) : this.$q.when({});
  }

  restoreDashboard(dashboard: DashboardModel, version: number) {
    const id = dashboard && dashboard.id ? dashboard.id : void 0;
    const url = `api/dashboards/db/${id}/restore`;
    return id && _.isNumber(version) ? this.backendSrv.post(url, { version }) : this.$q.when({});
  }
}

coreModule.service('auditSrv', AuditSrv);
