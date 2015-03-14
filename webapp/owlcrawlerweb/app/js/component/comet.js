define(function (require) {

  'use strict';

  /**
   * Module dependencies
   */

  var defineComponent = require('flight/lib/component');

  /**
   * Module exports
   */

  return defineComponent(comet);

  /**
   * Module function
   */

  function comet() {
    this.defaultAttrs({

    });

    this.startLongPool = function (_, payload) {
      var self = this;
      var delay = payload.delay;
      var pageId = payload.pageId;
      var index = payload.index;
      var cometId = payload.cometId;
      setTimeout(function(){
        $.ajax({
          url: '/api/comet?page=' + pageId + '&index=' + index + '&cometid=' + cometId,
          success: function(data){
            self.trigger('start-long-pool', {
              delay: 0,
              pageId: pageId,
              index: data.lastIndex,
              cometId: cometId
            });
            $(document).trigger(data.event, {
              message: data,
              prepend: false
            });
          },
          dataType: 'json',
          timeout: 120000 ,
          error: function(){
            self.trigger('start-long-pool', {
              delay: delay + 1000,
              pageId: pageId,
              index: index,
              cometId: cometId
            });
          }
        });
      },delay);
    };

    this.after('initialize', function () {
      this.on('start-long-pool', this.startLongPool);
      this.trigger('start-long-pool', {
        delay: 0,
        pageId: Math.random().toString(36).substring(7),
        index: window.index,
        cometId: window.cometId
      });
    });
  }

});
