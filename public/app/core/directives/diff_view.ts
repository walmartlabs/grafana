///<reference path="../../headers/common.d.ts" />

import coreModule from '../core_module';

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

coreModule.directive('compile', compile);
