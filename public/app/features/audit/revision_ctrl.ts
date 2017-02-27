///<reference path="../../headers/common.d.ts" />

import coreModule from 'app/core/core_module';

import {DashboardModel} from '../dashboard/model';

export interface CompareRevisionsModel {
  [key: number]: boolean;
}

export interface RevisionsModel {
  id: number;
  dashboardId: number;
  slug: string;
  version: number;
  created: Date;
  createdBy: number;
  message: string;
}

export class AuditLogCtrl {
  active: CompareRevisionsModel[];
  dashboard: DashboardModel;
  mode: string;
  revisions: RevisionsModel[];

  /** @ngInject */
  constructor(private $scope, private auditSrv) {
    $scope.ctrl = this;

    this.mode = 'list';
    this.dashboard = $scope.dashboard;
    this.active = [];
    this.reset();

    $scope.$watch('ctrl.mode', newVal => {
      if (newVal === 'compare') {
        this.compare();
      }
    });
  }

  auditLogChange() {
    return this.auditSrv.getAuditLog(this.dashboard).then(revisions => {
      this.revisions = revisions.reverse();
      this.active = this.revisions.map(revision => {
        return { [revision.version]: false };
      });
    });
  }

  reset() {
    this.auditLogChange();
  }

  compare() {
    console.log('compare', this.dashboard);
  }
}

coreModule.controller('AuditLogCtrl', AuditLogCtrl);
