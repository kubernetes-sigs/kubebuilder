function createHeading(style, title) {
  var icon = '';
  var title = '<div class="panel-title">' + title + '</div>';

  if (style != 'default') {
    icon = '<div class="panel-icon"><i class="icon-' + style + '"></i></div>';
  }

  return '<div class="panel-heading">' + icon + title  + '</div>';
}

function createPanel(style, title, content) {
  var element = '<div class="panel panel-' + style + '">';

  if (title) {
    element += createHeading(style, title);
  }

  return element + '<div class="panel-content">' + content + '</div></div>';
}

module.exports = {

  book: {
    assets: './assets',
    css: [
      'icons.css',
      'panel.css'
    ]
  },

  blocks: {
    panel: {
      process: function(block) {
        var style = block.kwargs.style || 'default';
        var title = block.kwargs.title;

        return this
            .renderBlock('markdown', block.body)
            .then(function(renderedBody) {
              return createPanel(style, title, renderedBody);
            });
      }
    }
  }

};
