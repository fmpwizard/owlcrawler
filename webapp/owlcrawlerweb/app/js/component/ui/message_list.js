define(function (require) {

  'use strict';

  /**
   * Module dependencies
   */

  var defineComponent = require('flight/lib/component');

  /**
   * Module exports
   */

  return defineComponent(messageList);

  /**
   * Module function
   */

  function messageList() {
    this.defaultAttrs({
      listSelector: '.f-message-row'
    });

    this.handleDataMessageSaved = function(event, payload){
      this.addMessageLine(payload.message, {append: true});
    };

    this.addMessageLine = function (payload, append) {
      var $messageRow = this.select('listSelector').first();
      var $clonedMessageRow = $messageRow.clone().removeAttr('id').removeClass('hidden');
      $clonedMessageRow.children('.f-message').first().text(payload.value.js);
      $clonedMessageRow.children('.f-time').first().text(payload.stamp); //TODO: add time here
      this.$node.append($clonedMessageRow);
    };

    this.handleDataMessages = function (event, payload) {
      if(payload.prepend){
        payload.message.resp.reverse().forEach(this.addMessageLine, this);
      } else {
        payload.message.resp.forEach(this.addMessageLine, this);
      }
      
    };

    this.after('initialize', function () {
      if ( this.$node.data('fetch-items') === true ) {
        this.trigger(document, 'uiNeedsMessages');
      }
      this.on(document, 'dataMessageSaved', this.handleDataMessageSaved);
      this.on(document, 'dataMessages', this.handleDataMessages);
    });
  }

});
