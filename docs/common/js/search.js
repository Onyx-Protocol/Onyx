$(document).ready(function(){
  // docgenerate returning undefined objects when creating search index files. TEMP FIX
  const steps = window.searchIndex.filter(function(n){ return n != undefined })

  var options = {
    shouldSort: true,
    includeMatches: true,
    includeScore: true,
    threshold: 0.25,
    tokenize: true,
    location: 0,
    distance: 100000,
    maxPatternLength: 32,
    matchAllTokens: true,
    minMatchCharLength: 3,
    keys: [{
      name: 'Title',
      weight: 0.8
    },{
      name: 'Snippet',
      weight: 0.6
    },{
      name: 'Body',
      weight: 0.6
    }]
  }
  var fuse = new Fuse(steps, options)
  var query = decodeURIComponent(window.location.search.slice(3).split('+').join(' '))
  var result = fuse.search(query.removeStopWords())

  $('#search-results').empty()
  $('#search-results').append('<div class="search-result">Search results for: <b>' + query + '</b></div>')
  result.some(function(res) {
    var obj = res['item']
    var url = obj['URL']
    var title = obj['Title']
    var snippet = obj['Snippet']
    if(title == ''){
      title = url.split('/').slice(-1)[0].replace(/-/g, ' ')
    }
    $('#search-results').append('<div class="search-result"><a href="' + url +'" target="blank">'+ title +'</a><br><span class="search-snippet">' + snippet + '</span></div>')
  })
})
