///<reference path="../../../headers/common.d.ts" />

import _ from 'lodash';
import moment from 'moment';

import coreModule from 'app/core/core_module';

import {DashboardModel} from '../model';

export interface RevisionsModel {
  id: number;
  checked: boolean;
  dashboardId: number;
  parentVersion: number;
  version: number;
  created: Date;
  createdBy: string;
  message: string;
}

export class AuditLogCtrl {
  dashboard: DashboardModel;
  delta: any;
  limit: number;
  loading: boolean;
  mode: string;
  revisions: RevisionsModel[];
  selected: number[];

  /** @ngInject */
  constructor(private $scope,
              private $rootScope,
              private $window,
              private contextSrv,
              private auditSrv) {
    $scope.ctrl = this;

    this.dashboard = $scope.dashboard;
    this.mode = 'list';
    this.limit = 2;
    this.selected = [];
    this.loading = false;

    this.resetFromSource();

    $scope.$watch('ctrl.mode', newVal => {
      $window.scrollTo(0, 0);
      if (newVal === 'list') {
        this.reset();
      }
    });
  }

  compareRevisionStateChanged(revision: any) {
    if (revision.checked) {
      this.selected.push(revision.version);
    } else {
      _.remove(this.selected, version => version === revision.version);
    }
  }

  compareRevisionDisabled(checked: boolean) {
    return this.selected.length === this.limit && !checked;
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
    return this.auditSrv.compareVersions(this.dashboard, this.selected).then(response => {
      this.delta = response;
    }).catch(err => {
      this.mode = 'list';
      this.$rootScope.appEvent('alert-error', ['There was an error fetching the diff', (err.message || err)]);
    }).finally(() => { this.loading = false; });
  }

  getLog() {
    this.loading = true;
    return this.auditSrv.getAuditLog(this.dashboard).then(revisions => {
      this.revisions = _.flow(
        _.partial(_.orderBy, _, rev => rev.version, 'desc'),
        _.partialRight(_.map, rev => _.extend({}, rev, { checked: false })),
      )(revisions);
    }).catch(err => {
      this.$rootScope.appEvent('alert-error', ['There was an error fetching the audit log', (err.message || err)]);
    }).finally(() => { this.loading = false; });
  }

  isComparable() {
    const isParamLength = this.selected.length === 2;
    const areNumbers = this.selected.every(version => _.isNumber(version));
    const areValidVersions = _.filter(this.revisions, revision => {
      return revision.version === this.selected[0] || revision.version === this.selected[1];
    }).length === 2;
    return isParamLength && areNumbers && areValidVersions;
  }

  reset() {
    this.delta = null;
    this.selected = [];
    this.revisions = _.map(this.revisions, rev => _.extend({}, rev, { checked: false }));
  }

  resetFromSource() {
    this.revisions = [];
    return this.getLog().then(this.reset.bind(this));
  }

  restore(version: number) {
    this.$rootScope.appEvent('confirm-modal', {
      title: 'Restore',
      text: 'Do you want to restore this dashboard?',
      text2: `The dashboard will be restored to version ${version}. All unsaved changes will be lost.`,
      icon: 'fa-rotate-right',
      yesText: 'Restore',
      onConfirm: this.restoreConfirm.bind(this, version),
    });
  }

  restoreConfirm(version: number) {
    this.loading = true;
    return this.auditSrv.restoreDashboard(this.dashboard, version).then(response => {
      this.revisions.unshift({
        id: this.revisions[0].id + 1,
        checked: false,
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
    }).catch(err => {
      this.$rootScope.appEvent('alert-error', ['There was an error restoring the dashboard', (err.message || err)]);
    }).finally(() => { this.loading = false; });
  }
}

coreModule.controller('AuditLogCtrl', AuditLogCtrl);
