///<reference path="../../headers/common.d.ts" />

import './revision_ctrl';

import coreModule from 'app/core/core_module';
import {DashboardModel} from '../dashboard/model';

export class AuditSrv {
  /** @ngInject */
  constructor(private backendSrv, private $q) {}

  getAuditLog(dashboard: DashboardModel) {
    const url = `api/dashboards/db/${dashboard.id}/versions`;
    return dashboard.id ? this.backendSrv.get(url) : this.$q.when([]);
  }

  compareVersions(dashboard: DashboardModel, compare: { original: number; new: number; }) {
    const url = `api/dashboards/db/${dashboard.id}/compare/${compare.original}...${compare.new}`;
    return this.backendSrv.get(url);
  }
}

coreModule.service('auditSrv', AuditSrv);
