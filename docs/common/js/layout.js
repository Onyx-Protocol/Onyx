"use strict";

function activateCurrentSidebarItem() {
	var p = location.pathname;
	p = p.replace(/\/+$/,"")
	var currentItem = $('.docs-nav a[href$="' + (p) + '"]')
 	currentItem.addClass('active');
	currentItem.parent().parent().addClass("show").show()
}

function prepareSidebarMenu() {
	$(".docs-nav").addClass("folded-by-default")

	activateCurrentSidebarItem()

    $(".toggle").click(function(e) {
      e.preventDefault();
      var sectionLink = $(this);
      if (sectionLink.next().hasClass('show')) {
        sectionLink.next().removeClass('show');
        sectionLink.next().slideUp(150);
      } else {
        sectionLink.parent().parent().find('li .inner').removeClass('show');
        sectionLink.parent().parent().find('li .inner').slideUp(150);
        sectionLink.next().toggleClass('show');
        sectionLink.next().slideToggle(150);
      }
    })
}

function prepareUpNextButton() {
	// 1. Figure which item in the sidebar is current.
	// 2. Find the next one.
	// 3. Take its link and title.
	// 4. Activate the next button.

	var p = location.pathname;
	p = p.replace(/\/+$/,"")

	if(p.split("/").slice(-1)[0] == "search-results") {
		return
	}
	
	var toc = $('.docs-nav .inner a')
	var currentIndex = -100

	var setUpNext = function(a, url, ci) {
		var upNext = $("#up-next")
		upNext.show()

		if (ci < 0) {
			$("h2", upNext).text($("h2", upNext).data("backtitle"))
		}

		var title = a.attr("title")
		if (!title || title == "") {
			title = a.text()
		}
		$("a > span", upNext).text(title)
		$("a", upNext).attr('href', url)
	}

	toc.each(function(i){
		var a = $(this)
		var url = a.attr("href")
		if (url == p) {
			currentIndex = i
			return
		}
		if (currentIndex >= 0 && i > currentIndex && !a.hasClass("skip-next-up")) {
			setUpNext(a, url, currentIndex)
			currentIndex = 9999; // reset this so we don't use the following items
			return
		}
	})

	if (currentIndex < 9999) {
		var a = toc.first()
		setUpNext(a, a.attr("href"), -1)
	}
}

function selectOSForDownload() {
	if ($("#download-options")[0]) {
		var ua = navigator.userAgent.toUpperCase()

		var isMac     = ua.indexOf('MAC') !== -1;
		var isWindows = ua.indexOf('WIN') !== -1;
		var isLinux   = ua.indexOf('LINUX') !== -1;

		if (isMac) {
			selectOSTab($('#download-option-mac a')[0], 'mac');
		} else if (isWindows) {
			selectOSTab($('#download-option-windows a')[0], 'windows');
		} else if (isLinux) {
			selectOSTab($('#download-option-linux a')[0], 'linux');
		} else {

		}
	}
}

// switcher between the navtabs for operating systems
function selectOSTab(target, osName) {
  // Declare all variables
  var i, tabcontent, tablinks;

  // Get all elements with class="tabcontent" and hide them
  tabcontent = document.getElementsByClassName("tabcontent");
  for (i = 0; i < tabcontent.length; i++) {
    tabcontent[i].style.display = "none";
  }

  // Get all elements with class="tablinks" and remove the class "active"
  tablinks = document.getElementsByClassName("tablinks");
  for (i = 0; i < tablinks.length; i++) {
    tablinks[i].className = tablinks[i].className.replace(" active", "");
  }

  // Show the current tab, and add an "active" class to the link that opened the tab
  document.getElementById(osName).style.display = "block";
  target.className += " active";
}

function attachSignupFormToDownloadButton() {
	$(".downloadBtn").click(function() {
	  	showSignUpForm();
	  	return true;
	});
}

// Modal to sign up for newsletter
function showSignUpForm() {
	 var modal = document.getElementById('downloadModal');

	// Make sure modal is in the body, not where it was originally deployed.
	$("body").append($(modal))

	// Get the button that opens the modal
	 var btn = document.getElementById("downloadBtn");

	// Get the <span> element that closes the modal
	 var span = document.getElementsByClassName("close")[0];

	// When the user clicks on the button, open the modal
	 modal.style.display = "block";

	// When the user clicks on <span> (x), close the modal
	 span.onclick = function () {
	   modal.style.display = "none";
	 }

	// When the user clicks anywhere outside of the modal, close it
	 window.onclick = function (event) {
	   if (event.target == modal) {
	     modal.style.display = "none";
	   }
	 }
}

function fixupSidenotes() {
	$(function(){
		// Sidenotes want to be logically under the paragraph to which they belong, but we want to render them to float right to them.
		// CSS only allows us to make sidenote float to the right of the *next* element, not the previous one.
		// To fix this, we simply relocate sidenote in runtime before it's preceding sibling.
		$(".sidenote").each(function(){
			var sidenote = $(this)
			sidenote.insertBefore(sidenote.prev())
		})
	})
}

function loadVersionOptions() {
	var currentVersion
	var matchedVersion = location.pathname.match('/docs/(\\d+\\.\\d+)/')
	if (matchedVersion) {
		currentVersion = matchedVersion[1]
	} else {
		// When running docs without a numeric prefix (i.e. local development)
		// hide the version selector and the legacy version alert.
		$('#version-select').remove()
		return
	}

	var versions = window.documentationVersions
	versions.sort().reverse().forEach(function(version) {
		var attributes = 'value = "' + version + '"'
		if (currentVersion == version) {
			attributes = attributes + ' selected'
		}
		var latest = version == versions[0] ? ' (latest)' : ''

		$('#version-select').append('<option ' + attributes + '>v' + version + latest + '</option>')
	})

	var prerelease = false
	if (!versions.includes(currentVersion)) {
		prerelease = true
		var attributes = 'selected value = "' + currentVersion + '"'
		$('#version-select').prepend('<option ' + attributes + '>v' + currentVersion + " (prerelease)" + '</option>')
	}

	if (versions[0] != currentVersion) {
		var alert = $('#version-alert')

		if (prerelease) {
			$('#version-alert').html('<p>You are viewing prerelease documentation.</p>')
		} else {
			$('.current', alert).text(currentVersion)
			$('.latest', alert).text(versions[0])
			$('.latest-link', alert).attr('href', window.location.href.replace(currentVersion, versions[0]))
		}
		$('#version-alert').show()
	}

	$('#version-select').on('change', function(e) {
		window.location = window.location.href.replace(currentVersion, $('#version-select')[0].value)
	})
}
