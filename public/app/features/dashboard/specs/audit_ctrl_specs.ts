import {describe, beforeEach, it, sinon, expect, angularMocks} from 'test/lib/common';

import { versions } from 'test/mocks/audit-mocks';
import {AuditLogCtrl} from 'app/features/dashboard/audit/audit_ctrl';
import config from 'app/core/config';

describe('AuditLogCtrl', function() {
  var ctx: any = {};
  var auditSrv: any = {
    getAuditLog: sinon.stub(),
  };

  var versionsResponse = versions();

  beforeEach(angularMocks.module('grafana.core'));
  beforeEach(angularMocks.module('grafana.services'));
  beforeEach(angularMocks.inject($rootScope => {
    ctx.scope = $rootScope.$new();
  }));

  describe('when the controller successfully loads the audit log', function() {
    beforeEach(angularMocks.inject(($controller, $q) => {
      auditSrv.getAuditLog.returns($q.when(versionsResponse));
      ctx.ctrl = $controller(AuditLogCtrl, {
        auditSrv,
        $scope: ctx.scope,
      });
      ctx.ctrl.$scope.$apply();
    }));

    it('should show the audit log', function() {
      expect(ctx.ctrl.mode).to.be('list');
    });

    it('should not show the loading indicator', function() {
      expect(ctx.ctrl.loading).to.be(false);
    });

    it('should reset the controller\'s state on load', function() {
      expect(ctx.ctrl.delta).to.be(null);
      expect(ctx.ctrl.compare.original).to.be(null);
      expect(ctx.ctrl.compare.new).to.be(null);
      expect(ctx.ctrl.loading).to.be(false);
      expect(ctx.ctrl.revisions).to.eql(versionsResponse.reverse());
    });

    it('should load the revisions sorted descending by version id', function() {
      // TODO: assert array max is head
      // TODO: assert array min is tail
    });
  });

  describe('when the controller has no audit log', function() {
    // TODO: tests when getAuditLog has an empty array as response
  });
});


