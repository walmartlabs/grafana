import {describe, beforeEach, it, sinon, expect, angularMocks} from 'test/lib/common';

import _ from 'lodash';
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
        expect(ctx.ctrl.selected.length).to.be(0);
        expect(ctx.ctrl.selected).to.eql([]);
      });

      it('should indicate loading has finished', function() {
        expect(ctx.ctrl.loading).to.be(false);
      });

      it('should store the revisions sorted desc by version id', function() {
        expect(ctx.ctrl.revisions[0].version).to.be(6);
        expect(ctx.ctrl.revisions[1].version).to.be(5);
        expect(ctx.ctrl.revisions[2].version).to.be(4);
      });

      it('should add a checked property to each revision', function() {
        var actual = _.filter(ctx.ctrl.revisions, rev => rev.hasOwnProperty('checked'));
        expect(actual.length).to.be(3);
      });

      it('should set all checked properties to false on reset', function() {
        ctx.ctrl.revisions[0].checked = true;
        ctx.ctrl.revisions[2].checked = true;
        ctx.ctrl.reset();
        var actual = _.filter(ctx.ctrl.revisions, rev => !rev.checked);
        expect(actual.length).to.be(3);
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
        expect(ctx.ctrl.selected.length).to.be(0);
        expect(ctx.ctrl.selected).to.eql([]);
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
      // []
      expect(ctx.ctrl.isComparable()).to.be(false);

      ctx.ctrl.selected = [6];
      expect(ctx.ctrl.isComparable()).to.be(false);

      ctx.ctrl.selected = [6, 4];
      expect(ctx.ctrl.isComparable()).to.be(true);

      ctx.ctrl.selected = [2, 1];
      expect(ctx.ctrl.isComparable()).to.be(false);
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

  describe('when the user wants to restore a revision', function() {
    var deferred;
    var auditSrv: any = {};
    var $rootScope: any = {};

    beforeEach(angularMocks.inject(($controller, $q) => {
      deferred = $q.defer();
      auditSrv.getAuditLog = sinon.stub().returns($q.when(versionsResponse));
      auditSrv.restoreDashboard = sinon.stub().returns(deferred.promise);
      $rootScope.appEvent = sinon.spy();
      ctx.ctrl = $controller(AuditLogCtrl, {
        auditSrv,
        contextSrv: { user: { name: 'Carlos' }},
        $rootScope,
        $scope: ctx.scope,
      });
      ctx.ctrl.dashboard = { id: 1 };
      ctx.ctrl.restore();
      ctx.ctrl.$scope.$apply();
    }));

    it('should display a modal allowing the user to restore or cancel', function() {
      expect($rootScope.appEvent.calledOnce).to.be(true);
      expect($rootScope.appEvent.calledWith('confirm-modal')).to.be(true);
    });

    describe('and restore is selected and successful', function() {
      beforeEach(function() {
        deferred.resolve(restoreResponse);
        ctx.ctrl.restoreConfirm(4);
        ctx.ctrl.$scope.$apply();
      });

      it('should indicate loading has finished', function() {
        expect(ctx.ctrl.loading).to.be(false);
      });

      it('should add an entry for the restored revision to the audit log', function() {
        expect(ctx.ctrl.revisions.length).to.be(4);
      });

      describe('the restored revision', function() {
        it('should have its id and version numbers incremented', function() {
          expect(ctx.ctrl.revisions[0].id).to.be(4);
          expect(ctx.ctrl.revisions[0].version).to.be(7);
        });

        // TODO: assert post-confirm state/behaviours
      });
    });

    describe('and restore fails to fetch', function() {
      beforeEach(function() {
        deferred.reject(new Error('RestoreError'));
        ctx.ctrl.restoreConfirm();
        ctx.ctrl.$scope.$apply();
      });

      it('should indicate loading has finished', function() {
        expect(ctx.ctrl.loading).to.be(false);
      });

      it('should broadcast an event indicating the failure', function() {
        expect($rootScope.appEvent.callCount).to.be(2);
        expect($rootScope.appEvent.getCall(0).calledWith('confirm-modal')).to.be(true);
        expect($rootScope.appEvent.getCall(1).calledWith('alert-error')).to.be(true);
        expect($rootScope.appEvent.getCall(1).args[1][0]).to.be('There was an error restoring the dashboard');
      });

      // TODO: test state after failure i.e. do we hide the modal or keep it visible
    });
  });
});


