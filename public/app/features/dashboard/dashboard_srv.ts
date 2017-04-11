///<reference path="../../headers/common.d.ts" />

import _ from 'lodash';
import coreModule from 'app/core/core_module';
import {DashboardModel} from './model';

export class DashboardSrv {
  dash: any;

  /** @ngInject */
  constructor(private backendSrv, private $rootScope, private $location) {
  }

  create(dashboard, meta) {
    return new DashboardModel(dashboard, meta);
  }

  setCurrent(dashboard) {
    this.dash = dashboard;
  }

  getCurrent() {
    return this.dash;
  }

  saveDashboard(options) {
    if (!this.dash.meta.canSave && options.makeEditable !== true) {
      return Promise.resolve();
    }

    if (this.dash.title === 'New dashboard') {
      return this.saveDashboardAs();
    }

    return this.saveDashboardMessage();
  }

  saveDashboardAs() {
    var newScope = this.$rootScope.$new();
    newScope.clone = this.dash.getSaveModelClone();
    newScope.clone.editable = true;
    newScope.clone.hideControls = false;

    this.$rootScope.appEvent('show-modal', {
      src: 'public/app/features/dashboard/partials/saveDashboardAs.html',
      scope: newScope,
      modalClass: 'modal--narrow'
    });
  }

  saveDashboardMessage() {
    var newScope = this.$rootScope.$new();
    newScope.clone = this.dash.getSaveModelClone();

    this.$rootScope.appEvent('show-modal', {
      src: 'public/app/features/dashboard/partials/saveDashboardMessage.html',
      scope: newScope,
      modalClass: 'modal--narrow'
    });
  }
}

coreModule.service('dashboardSrv', DashboardSrv);

