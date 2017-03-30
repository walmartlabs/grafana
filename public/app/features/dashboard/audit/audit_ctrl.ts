///<reference path="../../../headers/common.d.ts" />

import _ from 'lodash';
import moment from 'moment';

import coreModule from 'app/core/core_module';

import {DashboardModel} from '../model';

export interface RevisionsModel {
  id: number;
  dashboardId: number;
  parentVersion: number;
  version: number;
  created: Date;
  createdBy: string;
  message: string;
}

export class AuditLogCtrl {
  compare: { original: number; new: number; };
  dashboard: DashboardModel;
  delta: any;
  loading: boolean;
  mode: string;
  revisions: RevisionsModel[];

  /** @ngInject */
  constructor(private $scope,
              private $rootScope,
              private $window,
              private contextSrv,
              private auditSrv) {
    $scope.ctrl = this;

    this.dashboard = $scope.dashboard;
    this.mode = 'list';
    this.loading = false;

    this.resetFromSource();

    $scope.$watch('ctrl.mode', newVal => {
      $window.scrollTo(0, 0);
      if (newVal === 'list') {
        this.reset();
      }
    });
  }

  formatDate(date) {
    date = moment.isMoment(date) ? date : moment(date);
    const format = 'YYYY-MM-DD HH:mm:ss';

    return this.dashboard.timezone === 'browser' ?
      moment(date).format(format) :
      moment.utc(date).format(format);
  }

  getDiff() {
    this.mode = 'compare';
    this.loading = true;
    return this.auditSrv.compareVersions(this.dashboard, this.compare).then(response => {
      this.delta = response;
    }).catch(err => {
      this.mode = 'list';
      this.$rootScope.appEvent('alert-error', ['There was an error fetching the diff', (err.message || err)]);
    }).finally(() => { this.loading = false; });
  }

  getLog() {
    this.loading = true;
    return this.auditSrv.getAuditLog(this.dashboard).then(revisions => {
      this.revisions = _.orderBy(revisions, rev => rev.version, 'desc');
    }).catch(err => {
      this.$rootScope.appEvent('alert-error', ['There was an error fetching the audit log', (err.message || err)]);
    }).finally(() => { this.loading = false; });
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
    this.compare = { original: null, new: null };
  }

  resetFromSource() {
    this.revisions = [];
    return this.getLog().then(this.reset.bind(this));
  }

  restore(version: number) {
    this.$scope.appEvent('confirm-modal', {
      title: 'Restore',
      text: 'Do you want to restore this dashboard?',
      text2: `The dashboard will be restored to version ${version}. All unsaved changes will be lost.`,
      icon: 'fa-rotate-right',
      yesText: 'Restore',
      onConfirm: () => {
        this.loading = true;
        return this.auditSrv.restoreDashboard(this.dashboard, version).then(response => {
          this.revisions.unshift({
            id: this.revisions[0].id + 1,
            dashboardId: this.dashboard.id,
            parentVersion: version,
            version: this.revisions[0].version + 1,
            created: new Date(),
            createdBy: this.contextSrv.user.name,
            message: '',
          });

          this.reset();
          const restoredData = response.dashboard;
          this.dashboard = restoredData.dashboard;
          this.dashboard.meta = restoredData.meta;
          this.$scope.setupDashboard(restoredData);
        }).finally(() => { this.loading = false; });
      }
    });
  }
}

coreModule.controller('AuditLogCtrl', AuditLogCtrl);
