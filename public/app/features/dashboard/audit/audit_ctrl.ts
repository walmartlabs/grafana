///<reference path="../../../headers/common.d.ts" />

import _ from 'lodash';
import angular from 'angular';
import moment from 'moment';

import coreModule from 'app/core/core_module';

import {DashboardModel} from '../model';
import {AuditLogOpts, RevisionsModel} from './models';

export class AuditLogCtrl {
  appending: boolean;
  dashboard: DashboardModel;
  delta: { basic: string; html: string; };
  diff: string;
  limit: number;
  loading: boolean;
  max: number;
  mode: string;
  orderBy: string;
  revisions: RevisionsModel[];
  selected: number[];
  start: number;

  /** @ngInject */
  constructor(private $scope,
              private $rootScope,
              private $window,
              private $q,
              private contextSrv,
              private auditSrv) {
    $scope.ctrl = this;

    this.appending = false;
    this.dashboard = $scope.dashboard;
    this.diff = 'basic';
    this.limit = 10;
    this.loading = false;
    this.max = 2;
    this.mode = 'list';
    this.orderBy = 'version';
    this.selected = [];
    this.start = 0;

    this.resetFromSource();

    $scope.$watch('ctrl.mode', newVal => {
      $window.scrollTo(0, 0);
      if (newVal === 'list') {
        this.reset();
      }
    });

    $rootScope.onAppEvent('dashboard-saved', this.onDashboardSaved.bind(this));
  }

  addToLog() {
    this.start = this.start + this.limit;
    this.getLog(true);
  }

  compareRevisionStateChanged(revision: any) {
    if (revision.checked) {
      this.selected.push(revision.version);
    } else {
      _.remove(this.selected, version => version === revision.version);
    }
    this.selected = _.sortBy(this.selected);
  }

  compareRevisionDisabled(checked: boolean) {
    return (this.selected.length === this.max && !checked) || this.revisions.length === 1;
  }

  formatDate(date) {
    date = moment.isMoment(date) ? date : moment(date);
    const format = 'YYYY-MM-DD HH:mm:ss';

    return this.dashboard.timezone === 'browser' ?
      moment(date).format(format) :
      moment.utc(date).format(format);
  }

  formatBasicDate(date) {
    const now = this.dashboard.timezone === 'browser' ?  moment() : moment.utc();
    const then = this.dashboard.timezone === 'browser' ?  moment(date) : moment.utc(date);
    return then.from(now);
  }

  getDiff(diff: string) {
    if (!this.isComparable()) { return; } // disable button but not tooltip

    this.diff = diff;
    this.mode = 'compare';
    this.loading = true;

    // instead of using lodash to find min/max we use the index
    // due to the array being sorted in ascending order
    const compare = {
      new: this.selected[1],
      original: this.selected[0],
    };

    if (this.delta[this.diff]) {
      this.loading = false;
      return this.$q.when(this.delta[this.diff]);;
    } else {
      return this.auditSrv.compareVersions(this.dashboard, compare, diff).then(response => {
        this.delta[this.diff] = response;
      }).catch(err => {
        this.mode = 'list';
        this.$rootScope.appEvent('alert-error', ['There was an error fetching the diff', (err.message || err)]);
      }).finally(() => { this.loading = false; });
    }
  }

  getLog(append = false) {
    this.loading = !append;
    this.appending = append;
    const options: AuditLogOpts = {
      limit: this.limit,
      start: this.start,
      orderBy: this.orderBy,
    };
    return this.auditSrv.getAuditLog(this.dashboard, options).then(revisions => {
      const formattedRevisions =  _.flow(
        _.partialRight(_.map, rev => _.extend({}, rev, {
          checked: false,
          message: (revision => {
            if (revision.message === '') {
              if (revision.version === 1) {
                return 'Dashboard\'s initial save';
              }

              if (revision.restoredFrom > 0) {
                return `Restored from version ${revision.restoredFrom}`;
              }

              if (revision.parentVersion === 0) {
                return 'Dashboard overwritten';
              }

              return 'Dashboard saved';
            }
            return revision.message;
          })(rev),
        })))(revisions);

      this.revisions = append ? this.revisions.concat(formattedRevisions) : formattedRevisions;
    }).catch(err => {
      this.$rootScope.appEvent('alert-error', ['There was an error fetching the audit log', (err.message || err)]);
    }).finally(() => {
      this.loading = false;
      this.appending = false;
    });
  }

  getMeta(version: number, property: string) {
    const revision = _.find(this.revisions, rev => rev.version === version);
    return revision[property];
  }

  isOriginalCurrent() {
    return this.selected[1] === this.dashboard.version;
  }

  isComparable() {
    const isParamLength = this.selected.length === 2;
    const areNumbers = this.selected.every(version => _.isNumber(version));
    const areValidVersions = _.filter(this.revisions, revision => {
      return revision.version === this.selected[0] || revision.version === this.selected[1];
    }).length === 2;
    return isParamLength && areNumbers && areValidVersions;
  }

  isLastPage() {
    return _.find(this.revisions, rev => rev.version === 1);
  }

  onDashboardSaved() {
    this.$rootScope.appEvent('hide-dash-editor');
  }

  reset() {
    this.delta = { basic: '', html: '' };
    this.diff = 'basic';
    this.mode = 'list';
    this.revisions = _.map(this.revisions, rev => _.extend({}, rev, { checked: false }));
    this.selected = [];
    this.start = 0;
  }

  resetFromSource() {
    this.revisions = [];
    return this.getLog().then(this.reset.bind(this));
  }

  restore(version: number) {
    this.$rootScope.appEvent('confirm-modal', {
      title: 'Restore version',
      text: '',
      text2: `Are you sure you want to restore the dashboard to version ${version}? All unsaved changes will be lost.`,
      icon: 'fa-rotate-right',
      yesText: `Yes, restore to version ${version}`,
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
        message: `Restored from version ${version}`,
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
