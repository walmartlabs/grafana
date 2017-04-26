define([
  'angular',
],
function (angular) {
  'use strict';

  var module = angular.module('grafana.controllers');

  module.controller('SaveDashboardMessageCtrl', function($scope, backendSrv, dashboardSrv) {

    $scope.init = function() {
      $scope.clone.message = '';
      $scope.clone.max = 64;
    };

    function saveDashboard(options) {
      options.message = $scope.clone.message;
      return backendSrv.saveDashboard($scope.clone, options)
        .then(dashboardSrv.postSave.bind(dashboardSrv, $scope.clone))
        .catch(dashboardSrv.handleSaveDashboardError.bind(dashboardSrv, $scope.clone))
        .finally(function() { $scope.dismiss(); });
    }

    $scope.saveVersion = function(isValid) {
      if (!isValid) { return; }
      saveDashboard({overwrite: false});
    };
  });

});

