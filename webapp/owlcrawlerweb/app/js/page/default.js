define(function (require) {

  'use strict';

  /**
   * Module dependencies
   */

  var SendMessage = require('component/ui/send_message');
  var MessageList = require('component/ui/message_list');
  var LoadMore    = require('component/ui/load_more');
  var ChatData    = require('component/data/chat_data');
  var Comet       = require('component/comet');

  /**
   * Module exports
   */

  return initialize;

  /**
   * Module function
   */

  function initialize() {
    ChatData.attachTo(document);
    MessageList.attachTo('#message-list');
    SendMessage.attachTo('.f-send-message');
    LoadMore.attachTo('#f-load-more');
    Comet.attachTo('#message-list');
  }
});
