///<reference path="../../headers/common.d.ts" />

import './revision_ctrl';

import coreModule from 'app/core/core_module';
import {DashboardModel} from '../dashboard/model';

export class AuditSrv {
  /** @ngInject */
  constructor(private backendSrv) {}

  getAuditLog(dashboard: DashboardModel) {
    return this.backendSrv.get(`api/dashboards/db/${dashboard.meta.slug}/versions`);
  }
}

coreModule.service('auditSrv', AuditSrv);
