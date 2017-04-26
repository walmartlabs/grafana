define([
  'angular',
],
function (angular) {
  'use strict';

  var module = angular.module('grafana.controllers');

  module.controller('SaveDashboardAsCtrl', function($scope, backendSrv, dashboardSrv) {

    $scope.init = function() {
      $scope.clone.id = null;
      $scope.clone.editable = true;
      $scope.clone.title = $scope.clone.title + " Copy";

      // remove alerts
      $scope.clone.rows.forEach(function(row) {
        row.panels.forEach(function(panel) {
          delete panel.alert;
        });
      });

      // remove auto update
      delete $scope.clone.autoUpdate;
    };

    $scope.keyDown = function (evt) {
      if (evt.keyCode === 13) {
        $scope.saveClone();
      }
    };

    $scope.saveClone = function() {
      return backendSrv.saveDashboard($scope.clone, {overwrite: false})
        .then(dashboardSrv.postSave.bind(dashboardSrv, $scope.clone))
        .catch(dashboardSrv.handleSaveDashboardError.bind(dashboardSrv, $scope.clone))
        .finally(function() { $scope.dismiss(); });
    };
  });

});
