import {describe, beforeEach, it, sinon, expect, angularMocks} from 'test/lib/common';

import {AuditLogCtrl} from 'app/features/dashboard/audit/audit_ctrl';
import { versions, compare, restore } from 'test/mocks/audit-mocks';
import config from 'app/core/config';

describe('AuditLogCtrl', function() {
  var ctx: any = {};
  var versionsResponse: any = versions();
  var compareResponse: any = compare();
  var restoreResponse: any = restore;

  beforeEach(angularMocks.module('grafana.core'));
  beforeEach(angularMocks.module('grafana.services'));
  beforeEach(angularMocks.inject($rootScope => {
    ctx.scope = $rootScope.$new();
  }));

  describe('when the audit log component is loaded', function() {
    var deferred;
    var auditSrv: any = {};
    var $rootScope: any = {};

    beforeEach(angularMocks.inject(($controller, $q) => {
      deferred = $q.defer();
      auditSrv.getAuditLog = sinon.stub().returns(deferred.promise);
      $rootScope.appEvent = sinon.spy();
      ctx.ctrl = $controller(AuditLogCtrl, {
        auditSrv,
        $rootScope,
        $scope: ctx.scope,
      });
    }));

    it('should immediately attempt to fetch the audit log', function() {
      expect(auditSrv.getAuditLog.calledOnce).to.be(true);
    });

    describe('and the audit log is successfully fetched', function() {
      beforeEach(function() {
        deferred.resolve(versionsResponse);
        ctx.ctrl.$scope.$apply();
      });

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

    describe('and fetching the audit log fails', function() {
      beforeEach(function() {
        deferred.reject(new Error('AuditLogError'));
        ctx.ctrl.$scope.$apply();
      });

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

  describe('when the user wants to compare two revisions', function() {
    var deferred;
    var auditSrv: any = {};
    var $rootScope: any = {};

    beforeEach(angularMocks.inject(($controller, $q) => {
      deferred = $q.defer();
      auditSrv.getAuditLog = sinon.stub().returns($q.when(versionsResponse));
      auditSrv.compareVersions = sinon.stub().returns(deferred.promise);
      $rootScope.appEvent = sinon.spy();
      ctx.ctrl = $controller(AuditLogCtrl, {
        auditSrv,
        $rootScope,
        $scope: ctx.scope,
      });
      ctx.ctrl.$scope.$apply();
    }));

    it('should have already fetched the audit log', function() {
      expect(auditSrv.getAuditLog.calledOnce).to.be(true);
      expect(ctx.ctrl.revisions.length).to.be.above(0);
    });

    it('should check that two valid versions are selected', function() {
      // TODO: test isComparable
    });

    describe('and the diff is successfully fetched', function() {
      beforeEach(function() {
        deferred.resolve(compareResponse);
        ctx.ctrl.getDiff();
        ctx.ctrl.$scope.$apply();
      });

      it('should fetch the diff if two valid versions are selected', function() {
        expect(auditSrv.compareVersions.calledOnce).to.be(true);
        expect(ctx.ctrl.delta).to.eql(compareResponse);
      });

      it('should set the diff view as active', function() {
        expect(ctx.ctrl.mode).to.be('compare');
      });

      it('should indicate loading has finished', function() {
        expect(ctx.ctrl.loading).to.be(false);
      });
    });

    describe('and fetching the diff fails', function() {
      beforeEach(function() {
        deferred.reject(new Error('DiffError'));
        ctx.ctrl.getDiff();
        ctx.ctrl.$scope.$apply();
      });

      it('should fetch the diff if two valid versions are selected', function() {
        expect(auditSrv.compareVersions.calledOnce).to.be(true);
        expect(ctx.ctrl.delta).to.be(null);
      });

      it('should return to the audit log view', function() {
        expect(ctx.ctrl.mode).to.be('list');
      });

      it('should indicate loading has finished', function() {
        expect(ctx.ctrl.loading).to.be(false);
      });

      it('should broadcast an event indicating the failure', function() {
        expect($rootScope.appEvent.calledOnce).to.be(true);
        expect($rootScope.appEvent.calledWith('alert-error')).to.be(true);
      });

      it('should have an empty delta/changeset', function() {
        expect(ctx.ctrl.delta).to.be(null);
      });
    });
  });
});


