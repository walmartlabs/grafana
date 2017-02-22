///<reference path="../../headers/common.d.ts" />

import coreModule from 'app/core/core_module';

export class AuditLogCtrl {
  mode: any;

  /** @ngInject */
  constructor(private $scope) {
    $scope.ctrl = this;

    this.mode = 'list';

    $scope.$watch('ctrl.mode', newVal => {
      if (newVal === 'compare') {
        this.compare();
      }
    });
  }

  compare() {
    console.log('compare');
  }
}

coreModule.controller('AuditLogCtrl', AuditLogCtrl);
