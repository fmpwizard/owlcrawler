define(function (require) {

  'use strict';

  /**
   * Module dependencies
   */

  var defineComponent = require('flight/lib/component');
  var withRestBackend = require('component/with_data/with_rest_backend');

  /**
   * Module exports
   */

  return defineComponent(chatData, withRestBackend);

  /**
   * Module function
   */

  function chatData() {
    this.defaultAttrs({

    });

    
    this.handleMessageSent = function (event, message) {
      var self = this;
      $.when(this.save(message.message))
        .then(function(data, status){
          message.message.id = data.id;
          var result = message;
          //We now let the comet component give us the new message.
          //self.trigger('dataMessageSaved', result);
        });
    };

    this.handleUiNeedsMessages = function (_, payload) {
      var lastPage = 0;
      if (payload && payload.message){
        lastPage = payload.message.lastPage || 0;
      }

      var self = this;
      $.when(this.getPaginated( lastPage ))
        .then(function(data, status){

          var dataAsArray = [];
          for (var key in data) {
            dataAsArray.push(data[key]);
          }
          self.trigger('dataMessages', {
            messages: dataAsArray,
            prepend: true
          });
          self.trigger(document, 'uiLastPage', {
            message: {lastPage: lastPage}
          });
        });
    };

    this.after('initialize', function () {
      this.on('uiMessageSent', this.handleMessageSent);
      this.on('uiNeedsMessages', this.handleUiNeedsMessages);
    });
  }

});
