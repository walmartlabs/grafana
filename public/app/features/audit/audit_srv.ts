///<reference path="../../headers/common.d.ts" />

import './revision_ctrl';

import coreModule from 'app/core/core_module';

export class AuditSrv {
  globalAuditPromise: any;

  /** @ngInject */
  constructor(private $rootScope) {
    $rootScope.onAppEvent('refresh', this.clearCache.bind(this), $rootScope);
  }

  clearCache() {
    this.globalAuditPromise = null;
  }
}

coreModule.service('auditSrv', AuditSrv);

