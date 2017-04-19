///<reference path="../../headers/common.d.ts" />

import _ from 'lodash';
import coreModule from '../core_module';

// Directive to compile fetched templates
function compile($compile) {
  return {
    restrict: 'A',
    link: function(scope, element, attrs) {
      scope.$watch(scope => scope.$eval(attrs.compile), value => {
        element.html(value);
        $compile(element.contents())(scope);
      });
    }
  };
}

// Container for a set of changes
export function list() {
  return {
    replace: true,
    restrict: 'E',
    scope: {
      name: '@name',
      changeType: '@type',
    },
    templateUrl: 'public/app/features/dashboard/audit/partials/diff-list.html',
    transclude: true,
  };
}

// Individual change
export class ListItemCtrl {
  name: string;

  /** @ngInject */
  constructor(private $scope) {
    this.name = _.startCase($scope.name);
  }
}

export function listItem() {
  return {
    controller: ListItemCtrl,
    controllerAs: 'ctrl',
    replace: true,
    restrict: 'E',
    scope: {
      name: '@',
      new: '@',
      original: '@',
      changeType: '@type',
    },
    templateUrl: 'public/app/features/dashboard/audit/partials/diff-item.html',
    transclude: true,
  };
}

coreModule.directive('compile', compile);
coreModule.directive('diffList', list);
coreModule.directive('diffListItem', listItem);
