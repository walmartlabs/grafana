define([
  'angular',
],
function (angular) {
  'use strict';

  var module = angular.module('grafana.controllers');

  module.controller('SaveDashboardMessageCtrl', function($scope, $location, backendSrv, dashboardSrv) {

    $scope.init = function() {
      $scope.clone.message = '';
      $scope.clone.max = 64;
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

    $scope.saveVersion = function(isValid) {
      if (!isValid) { return; }

      saveDashboard({overwrite: false}).then(null, function(err) {
        if (err.data && err.data.status === "version-mismatch") {
          err.isHandled = true;

          $scope.appEvent('confirm-modal', {
            title: 'Conflict',
            text: 'Someone else has updated this dashboard.',
            text2: 'Would you still like to save this dashboard?',
            yesText: "Save & Overwrite",
            icon: 'fa-warning',
            onConfirm: function() {
              saveDashboard({overwrite: true});
            }
          });
        }

        if (err.data && err.data.status === "name-exists") {
          err.isHandled = true;

          $scope.appEvent('confirm-modal', {
            title: 'Conflict',
            text: 'Dashboard with the same name exists.',
            text2: 'Would you still like to save this dashboard?',
            yesText: "Save & Overwrite",
            icon: 'fa-warning',
            onConfirm: function() {
              saveDashboard({overwrite: true});
            }
          });
        }

        if (err.data && err.data.status === "plugin-dashboard") {
          err.isHandled = true;

          $scope.appEvent('confirm-modal', {
            title: 'Plugin Dashboard',
            text: err.data.message,
            text2: 'Your changes will be lost when you update the plugin. Use Save As to create custom version.',
            yesText: 'Overwrite',
            icon: 'fa-warning',
            altActionText: 'Save As',
            onAltAction: function() {
              dashboardSrv.saveDashboardAs();
            },
            onConfirm: function() {
              saveDashboard({overwrite: true});
            }
          });
        }
      });
    };
  });

});

