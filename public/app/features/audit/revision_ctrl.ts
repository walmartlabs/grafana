///<reference path="../../headers/common.d.ts" />

import _ from 'lodash';
import coreModule from 'app/core/core_module';

import {DashboardModel} from '../dashboard/model';

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
  old: number;
  new: number;
  dashboard: DashboardModel;
  mode: string;
  revisions: RevisionsModel[];

  /** @ngInject */
  constructor(private $scope, private auditSrv) {
    $scope.ctrl = this;

    this.mode = 'list';
    this.dashboard = $scope.dashboard;
    this.old = null;
    this.new = null;

    this.reset();

    $scope.$watch('ctrl.mode', newVal => {
      if (newVal === 'restore') {
        this.reset();
      }
    });
  }

  auditLogChange() {
    return this.auditSrv.getAuditLog(this.dashboard).then(revisions => {
      this.revisions = revisions.reverse();
    });
  }

  isComparable() {
    const areNumbers = _.isNumber(this.old) && _.isNumber(this.new);
    const areValidVersions = _.filter(this.revisions, revision => {
      return revision.version === this.old || revision.version === this.new;
    }).length === 2;
    return areNumbers && areValidVersions;
  }

  compare() {
    this.mode = 'compare';
    console.log('comparing %o version %d to version %d', this.dashboard, this.old, this.new);
  }

  reset() {
    this.auditLogChange();
  }
}

coreModule.controller('AuditLogCtrl', AuditLogCtrl);
