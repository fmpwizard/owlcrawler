'use strict';

requirejs.config({
  baseUrl: '',
  paths: {
    'flight': '../bower_components/flight',
    'component': '../js/component',
    'page': '../js/page',
    'jquery': '../bower_components/jquery/dist',
    'components-bootstrap': '../bower_components/components-bootstrap/js'
  }
});

require(
  [
    'flight/lib/compose',
    'flight/lib/registry',
    'flight/lib/advice',
    'flight/lib/logger',
    'flight/lib/debug',
    'jquery/jquery',
    'components-bootstrap/bootstrap',
    'page/default'
  ],

  function(compose, registry, advice, withLogging, debug) {
    debug.enable(true);
    compose.mixin(registry, [advice.withAdvice, withLogging]);

    require(['page/default'], function(initializeDefault) {
      initializeDefault();
    });
  }
);
