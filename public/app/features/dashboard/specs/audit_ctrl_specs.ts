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

  describe('when the controller successfully fetches the audit log', function() {
    beforeEach(angularMocks.inject(($controller, $q) => {
      auditSrv.getAuditLog.returns($q.when(versionsResponse));
      ctx.ctrl = $controller(AuditLogCtrl, {
        auditSrv,
        $scope: ctx.scope,
      });
      ctx.ctrl.$scope.$apply();
    }));

    it('should reset the controller\'s state', function() {
      expect(ctx.ctrl.mode).to.be('list');
      expect(ctx.ctrl.delta).to.be(null);
      expect(ctx.ctrl.compare.original).to.be(null);
      expect(ctx.ctrl.compare.new).to.be(null);
    });

    it('should indicate loading has finished', function() {
      expect(ctx.ctrl.loading).to.be(false);
    });

    it('should store the revisions sorted desc by version id', function() {
      expect(ctx.ctrl.revisions[0].version).to.be(6);
      expect(ctx.ctrl.revisions[1].version).to.be(5);
      expect(ctx.ctrl.revisions[2].version).to.be(4);
    });
  });

  describe('when the controller has no audit log', function() {
    var $rootScope;
    beforeEach(angularMocks.inject(($controller, $q) => {
      $rootScope = { appEvent: sinon.spy() };
      auditSrv.getAuditLog.returns($q.reject(new Error('test')));
      ctx.ctrl = $controller(AuditLogCtrl, {
        auditSrv,
        $rootScope,
        $scope: ctx.scope,
      });
      ctx.ctrl.$scope.$apply();
    }));

    it('should reset the controller\'s state', function() {
      expect(ctx.ctrl.mode).to.be('list');
      expect(ctx.ctrl.delta).to.be(null);
      expect(ctx.ctrl.compare.original).to.be(null);
      expect(ctx.ctrl.compare.new).to.be(null);
    });

    it('should indicate loading has finished', function() {
      expect(ctx.ctrl.loading).to.be(false);
    });

    it('should broadcast an event indicating the failure', function() {
      expect($rootScope.appEvent.calledOnce).to.be(true);
      expect($rootScope.appEvent.calledWith('alert-error')).to.be(true);
    });

    it('should have an empty revisions list', function() {
      expect(ctx.ctrl.revisions).to.eql([]);
    });
  });
});


