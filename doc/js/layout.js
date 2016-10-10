
"use strict";

$(function() {
    if (location.pathname == '/')  {
        $('.docs-nav a[href$="'+('/install')+'"]').addClass('active');
    } else {
        var p = location.pathname;
        $('.docs-nav a[href$="'+(p)+'"]').addClass('active');
    }
});

