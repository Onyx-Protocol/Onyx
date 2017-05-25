function updateSelection(lang) {
  try {
    localStorage.setItem('docsSelectedLang', lang)
  } catch (err) { /* no local storage */ }

  $('.snippet-set pre:not(.' + lang + ')').hide()
  $('.snippet-set pre.' + lang).show()
  $('.snippet-set [data-docs-lang]').removeClass('selected')
  $('.snippet-set [data-docs-lang="' + lang + '"]').addClass('selected')
}

$(function() {
  var allowed = ['java', 'js', 'rb']

  var initial
  try {
    initial = localStorage.getItem('docsSelectedLang')
  } catch (err) { /* no local storage */ }

  if (allowed.indexOf(initial) < 0) {
    initial = 'java'
  }

  updateSelection(initial)

  $(document).on('click', '[data-docs-lang]', function(event) {
    updateSelection($(event.target).data('docs-lang'))
  })
})
