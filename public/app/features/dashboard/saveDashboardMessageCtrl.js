define([
  'angular',
],
function (angular) {
  'use strict';

  var module = angular.module('grafana.controllers');

  module.controller('SaveDashboardMessageCtrl', function($scope, backendSrv, $location) {

    $scope.init = function() {
      $scope.clone.message = '';
    };

    function saveDashboard(options) {
      options.message = $scope.clone.message;
      return backendSrv.saveDashboard($scope.clone, options).then(function(result) {
        var version = $scope.clone.version + 1;
        $scope.appEvent('alert-success', ['Dashboard saved', 'Version ' + version + ' saved in changelog']);

        $location.url('/dashboard/db/' + result.slug);

        $scope.appEvent('dashboard-saved', $scope.clone);
        $scope.dismiss();
      });
    }

    $scope.saveVersion = function() {
      saveDashboard({overwrite: false}).then(null, function(err) {
        if (err.data && err.data.status === "name-exists") {
          err.isHandled = true;

          $scope.appEvent('confirm-modal', {
            title: 'Conflict',
            text: 'Dashboard with the same name exists.',
            text2: 'Would you still like to save this dashboard?',
            yesText: "Save & Overwrite",
            icon: "fa-warning",
            onConfirm: function() {
              saveDashboard({overwrite: true});
            }
          });
        }
      });
    };
  });

});

