define(function (require) {

  'use strict';

  /**
   * Module dependencies
   */

  var defineComponent = require('flight/lib/component');

  /**
   * Module exports
   */

  return defineComponent(loadMore);

  /**
   * Module function
   */

  function loadMore() {
    this.defaultAttrs({
      loadMoreSelector: '#f-load-more',
    });

    this.handleLoadMore = function (event) {
      event.preventDefault();
      var lastPage = this.$node.data('page');
      this.trigger(document, 'uiNeedsMessages', {
        message: {lastPage: lastPage + 1}
      });
    };

    this.handleUiLastPage = function  (_, payload) {
      this.$node.data('page', payload.message.lastPage);
    };

    this.after('initialize', function () {
      this.on('click', this.handleLoadMore);
      this.on(document, 'uiLastPage', this.handleUiLastPage);
    });
  }

});
