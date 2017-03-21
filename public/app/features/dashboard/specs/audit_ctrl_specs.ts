import {describe, beforeEach, it, sinon, expect, angularMocks} from 'test/lib/common';

import {AuditLogCtrl} from 'app/features/dashboard/audit/audit_ctrl';
import config from 'app/core/config';

describe('AuditLogCtrl', function() {
  var ctx: any = {};

  beforeEach(angularMocks.module('grafana.core'));
  beforeEach(angularMocks.module('grafana.services'));

  beforeEach(angularMocks.inject(($rootScope, $controller) => {
    ctx.scope = $rootScope.$new();
    ctx.ctrl = $controller(AuditLogCtrl, {
      $scope: ctx.scope,
    });
  }));


  describe('when the controller is loaded', function() {
    it('should show the audit log', function() {
      expect(ctx.ctrl.mode).to.eql('list');
    });
  });
});


