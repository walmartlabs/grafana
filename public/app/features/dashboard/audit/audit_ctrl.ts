///<reference path="../../../headers/common.d.ts" />

import _ from 'lodash';
import coreModule from 'app/core/core_module';

import {DashboardModel} from '../model';

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
  compare: { original: number; new: number; };
  dashboard: DashboardModel;
  delta: any;
  mode: string;
  revisions: RevisionsModel[];

  /** @ngInject */
  constructor(private $scope, private auditSrv) {
    $scope.ctrl = this;

    this.dashboard = $scope.dashboard;
    this.mode = 'list';

    this.resetFromSource();

    $scope.$watch('ctrl.mode', newVal => {
      if (newVal === 'list') {
        this.reset();
      }
    });
  }

  getLog() {
    return this.auditSrv.getAuditLog(this.dashboard).then(revisions => {
      this.revisions = revisions.reverse();
    });
  }

  getDiff() {
    this.mode = 'compare';
    return this.auditSrv.compareVersions(this.dashboard, this.compare).then(response => {
      this.delta = response;
    });
  }

  isComparable() {
    const c = this.compare;
    const areNumbers = _.isNumber(c.original) && _.isNumber(c.new);
    const areValidVersions = _.filter(this.revisions, revision => {
      return revision.version === c.original || revision.version === c.new;
    }).length === 2;
    return areNumbers && areValidVersions;
  }

  reset() {
    this.delta = null;
  }

  resetFromSource() {
    this.reset();
    this.revisions = [];
    this.compare = { original: null, new: null };
    this.getLog();
  }
}

coreModule.controller('AuditLogCtrl', AuditLogCtrl);
