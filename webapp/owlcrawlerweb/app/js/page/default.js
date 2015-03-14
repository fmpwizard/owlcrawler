define(function (require) {

  'use strict';

  /**
   * Module dependencies
   */

  var ChatData    = require('component/data/chat_data');

  /**
   * Module exports
   */

  return initialize;

  /**
   * Module function
   */

  function initialize() {
    ChatData.attachTo(document);
  }
});
