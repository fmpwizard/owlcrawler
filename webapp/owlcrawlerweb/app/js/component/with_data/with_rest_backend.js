define(function (require) {

  'use strict';

  /**
   * Module exports
   */

  return withRestBackend;

  /**
   * Module function
   */

  function withRestBackend() {
    this.defaultAttrs({
    });

    this.save = function( message ){
      var result = $.ajax({
        type: 'GET',
        contentType: 'application/json',
        url: '/api/search?term='+message
      });

      return result;

    };

    this.getPaginated = function (lastPage) {
      var result = $.ajax({
        type: 'GET',
        contentType: 'application/json',
        url: '/api/messages/page/'+lastPage
      });
      return result;
    };
  }
});
