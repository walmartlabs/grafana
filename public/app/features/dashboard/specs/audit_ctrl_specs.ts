import {describe, beforeEach, it, sinon, expect, angularMocks} from 'test/lib/common';

import _ from 'lodash';
import {AuditLogCtrl} from 'app/features/dashboard/audit/audit_ctrl';
import { versions, compare, restore } from 'test/mocks/audit-mocks';
import config from 'app/core/config';

describe('AuditLogCtrl', function() {
  var RESTORE_ID = 4;

  var ctx: any = {};
  var versionsResponse: any = versions();
  var compareResponse: any = compare();
  var restoreResponse: any = restore(7, RESTORE_ID);

  beforeEach(angularMocks.module('grafana.core'));
  beforeEach(angularMocks.module('grafana.services'));
  beforeEach(angularMocks.inject($rootScope => {
    ctx.scope = $rootScope.$new();
  }));

  var auditSrv;
  var $rootScope;
  beforeEach(function() {
    auditSrv = {
      getAuditLog: sinon.stub(),
      compareVersions: sinon.stub(),
      restoreDashboard: sinon.stub(),
    };
    $rootScope = {
      appEvent: sinon.spy(),
      onAppEvent: sinon.spy(),
    };
  });

  describe('when the audit log component is loaded', function() {
    var deferred;

    beforeEach(angularMocks.inject(($controller, $q) => {
      deferred = $q.defer();
      auditSrv.getAuditLog.returns(deferred.promise);
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
        expect(ctx.ctrl.delta).to.be('');
        expect(ctx.ctrl.selected.length).to.be(0);
        expect(ctx.ctrl.selected).to.eql([]);
        expect(_.find(ctx.ctrl.revisions, rev => rev.checked)).to.be.undefined;
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
        ctx.ctrl.selected = [0, 2];
        ctx.ctrl.reset();
        var actual = _.filter(ctx.ctrl.revisions, rev => !rev.checked);
        expect(actual.length).to.be(3);
        expect(ctx.ctrl.selected).to.eql([]);
      });
    });

    describe('and fetching the audit log fails', function() {
      beforeEach(function() {
        deferred.reject(new Error('AuditLogError'));
        ctx.ctrl.$scope.$apply();
      });

      it('should reset the controller\'s state', function() {
        expect(ctx.ctrl.mode).to.be('list');
        expect(ctx.ctrl.delta).to.be('');
        expect(ctx.ctrl.selected.length).to.be(0);
        expect(ctx.ctrl.selected).to.eql([]);
        expect(_.find(ctx.ctrl.revisions, rev => rev.checked)).to.be.undefined;
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

    describe('should update the audit log when the dashboard is saved', function() {
      beforeEach(function() {
        ctx.ctrl.dashboard = { version: 3 };
        ctx.ctrl.resetFromSource = sinon.spy();
      });

      it('should listen for the `dashboard-saved` appEvent', function() {
        expect($rootScope.onAppEvent.calledOnce).to.be(true);
        expect($rootScope.onAppEvent.getCall(0).args[0]).to.be('dashboard-saved');
      });

      it('should call `onDashboardSaved` when the appEvent is received', function() {
        expect($rootScope.onAppEvent.getCall(0).args[1]).to.not.be(ctx.ctrl.onDashboardSaved);
        expect($rootScope.onAppEvent.getCall(0).args[1].toString).to.be(ctx.ctrl.onDashboardSaved.toString);
      });

      it('should emit an appEvent to hide the changelog', function() {
        ctx.ctrl.onDashboardSaved();
        expect($rootScope.appEvent.calledOnce).to.be(true);
        expect($rootScope.appEvent.getCall(0).args[0]).to.be('hide-dash-editor');
      });
    });
  });

  describe('when the user wants to compare two revisions', function() {
    var deferred;

    beforeEach(angularMocks.inject(($controller, $q) => {
      deferred = $q.defer();
      auditSrv.getAuditLog.returns($q.when(versionsResponse));
      auditSrv.compareVersions.returns(deferred.promise);
      ctx.ctrl = $controller(AuditLogCtrl, {
        auditSrv,
        $rootScope,
        $scope: ctx.scope,
      });
      ctx.ctrl.$scope.onDashboardSaved = sinon.spy();
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
        ctx.ctrl.selected = [6, 4];
        ctx.ctrl.getDiff('basic');
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
        ctx.ctrl.selected = [6, 4];
        ctx.ctrl.getDiff('basic');
        ctx.ctrl.$scope.$apply();
      });

      it('should fetch the diff if two valid versions are selected', function() {
        expect(auditSrv.compareVersions.calledOnce).to.be(true);
        expect(ctx.ctrl.delta).to.be('');
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
        expect(ctx.ctrl.delta).to.be('');
      });
    });
  });

  describe('when the user wants to restore a revision', function() {
    var deferred;

    beforeEach(angularMocks.inject(($controller, $q) => {
      deferred = $q.defer();
      auditSrv.getAuditLog.returns($q.when(versionsResponse));
      auditSrv.restoreDashboard.returns(deferred.promise);
      ctx.ctrl = $controller(AuditLogCtrl, {
        auditSrv,
        contextSrv: { user: { name: 'Carlos' }},
        $rootScope,
        $scope: ctx.scope,
      });
      ctx.ctrl.$scope.setupDashboard = sinon.stub();
      ctx.ctrl.dashboard = { id: 1 };
      ctx.ctrl.restore();
      ctx.ctrl.$scope.$apply();
    }));

    it('should display a modal allowing the user to restore or cancel', function() {
      expect($rootScope.appEvent.calledOnce).to.be(true);
      expect($rootScope.appEvent.calledWith('confirm-modal')).to.be(true);
    });

    describe('from the diff view', function() {
      it('should return to the list view on restore', function() {
        ctx.ctrl.mode = 'compare';
        deferred.resolve(restoreResponse);
        ctx.ctrl.restoreConfirm(RESTORE_ID);
        ctx.ctrl.$scope.$apply();
        expect(ctx.ctrl.mode).to.be('list');
      });
    });

    describe('and restore is selected and successful', function() {
      beforeEach(function() {
        deferred.resolve(restoreResponse);
        ctx.ctrl.restoreConfirm(RESTORE_ID);
        ctx.ctrl.$scope.$apply();
      });

      it('should indicate loading has finished', function() {
        expect(ctx.ctrl.loading).to.be(false);
      });

      it('should add an entry for the restored revision to the audit log', function() {
        expect(ctx.ctrl.revisions.length).to.be(4);
      });

      describe('the restored revision', function() {
        var first;
        beforeEach(function() { first = ctx.ctrl.revisions[0]; });

        it('should have its `id` and `version` numbers incremented', function() {
          expect(first.id).to.be(4);
          expect(first.version).to.be(7);
        });

        it('should set `parentVersion` to the reverted version', function() {
          expect(first.parentVersion).to.be(RESTORE_ID);
        });

        it('should set `dashboardId` to the dashboard\'s id', function() {
          expect(first.dashboardId).to.be(1);
        });

        it('should set `created` to date to the current time', function() {
          expect(_.isDate(first.created)).to.be(true);
        });

        it('should set `createdBy` to the username of the user who reverted', function() {
          expect(first.createdBy).to.be('Carlos');
        });

        it('should set `message` to the user\'s commit message', function() {
          expect(first.message).to.be('Restored from version 4');
        });
      });

      it('should reset the controller\'s state', function() {
        expect(ctx.ctrl.mode).to.be('list');
        expect(ctx.ctrl.delta).to.be('');
        expect(ctx.ctrl.selected.length).to.be(0);
        expect(ctx.ctrl.selected).to.eql([]);
        expect(_.find(ctx.ctrl.revisions, rev => rev.checked)).to.be.undefined;
      });

      it('should set the dashboard object to the response dashboard data', function() {
        expect(ctx.ctrl.dashboard).to.eql(restoreResponse.dashboard.dashboard);
        expect(ctx.ctrl.dashboard.meta).to.eql(restoreResponse.dashboard.meta);
      });

      it('should call setupDashboard to render new revision', function() {
        expect(ctx.ctrl.$scope.setupDashboard.calledOnce).to.be(true);
        expect(ctx.ctrl.$scope.setupDashboard.getCall(0).args[0]).to.eql(restoreResponse.dashboard);
      });
    });

    describe('and restore fails to fetch', function() {
      beforeEach(function() {
        deferred.reject(new Error('RestoreError'));
        ctx.ctrl.restoreConfirm(RESTORE_ID);
        ctx.ctrl.$scope.$apply();
      });

      it('should indicate loading has finished', function() {
        expect(ctx.ctrl.loading).to.be(false);
      });

      it('should broadcast an event indicating the failure', function() {
        expect($rootScope.appEvent.callCount).to.be(2);
        expect($rootScope.appEvent.getCall(0).calledWith('confirm-modal')).to.be(true);
        expect($rootScope.appEvent.getCall(1).args[0]).to.be('alert-error');
        expect($rootScope.appEvent.getCall(1).args[1][0]).to.be('There was an error restoring the dashboard');
      });

      // TODO: test state after failure i.e. do we hide the modal or keep it visible
    });
  });
});


