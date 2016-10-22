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


//function adjustWidths() {
//	setTimeout(function(){
//		$("#doc-content > table").each(function(){
//			if ($(this).width() > 680) {
//				$(this).css("width", "100%")
//			}
//		})
//		$("#doc-content > pre").each(function(){
//			if ($("code", $(this))[0].scrollWidth > 680) {
//				$(this).css("width", "100%")
//			}
//		})
//	},100)
//}

//function adjustWidths() {
//	setTimeout(function(){
//		$("#doc-content > table").each(function(){
//				$(this).css("width", "770px")
//
//		})
//		$("#doc-content > pre").each(function(){
//				$(this).css("width", "770px")
//		})
//	},100)
//}

// This finds <p> <div class="sidenote"> pairs and wraps them in <section class="text-block">.
// Inner paragraphs are wrapped together in <div class="text-main">.
// The result is:
/*
	<p>
	<p>
	...
	<section class="text-block">
		<div class="text-main">
			<p>
			<p>
			...
		</div>
		<div class="sidenote">
			<p>
			<p>
			...
		</div>
	</section>
	...
	<p>

*/

// function rewrapSidenotes() {
// 	var collectedElements = {
// 		state: "idle",
// 		paragraphs: [],
// 		sidenote: null
// 	}
// 	$(".sidenotes-container > *").each(function(index){
// 		var element = $(this)
// 		var klass = element.attr('class')
// 		var tag = element[0].nodeName.toLowerCase()
// 
// 		//console.log("tag: " + element[0].nodeName.toLowerCase())
// 
// 		if (collectedElements.state == "idle") {
// 			if (klass != "sidenote") {
// 				// We need to find the first element before the sidenote.
// 				// So we remember one element only, right before the sidenote.
// 				collectedElements.paragraphs = [ element ]
// 			} else {
// 				// If this is an sidenote, start collecting paragraphs
// 				collectedElements.sidenote = element
// 				collectedElements.state = "collecting"
// 
// 				//console.log("Begin collecting")
// 			}
// 		} else if (collectedElements.state == "collecting") {
// 			if (klass != "sidenote") {
// 				// Lets remember all elements until we find the sidenote.
// 				// If/when we find the sidenote, we'll pop that element.
// 				//console.log("collecting: adding an element: " + tag)
// 				collectedElements.paragraphs.push(element)
// 			} else {
// 				// Oops, some other sidenote is detected - let's forget the last added paragraph - it belongs to that sidenote's group.
// 				var lastparagraph = collectedElements.paragraphs.pop()
// 				var nextsidenote = element
// 
// 				// 1. Wrap collected elements.
// 				// 1.1. Insert sidenote after the last paragraph.
// 				//console.log("1.1. Insert after the last paragraph")
// 				collectedElements.sidenote.insertAfter(collectedElements.paragraphs[collectedElements.paragraphs.length - 1])
// 
// 				// 1.2. Wrap all paragraphs.
// 				//console.log("1.2. Wrap all paragraphs")
// 				$.each(collectedElements.paragraphs, function(index, value) {
// 					value.addClass("temp-wrapping-class")
// 				});
// 
// 				$(".temp-wrapping-class").wrapAll("<div class=\"text-main temp-superwrapping-class\"></div>")
// 
// 				$.each(collectedElements.paragraphs, function(index, value) {
// 					value.removeClass("temp-wrapping-class")
// 				});
// 
// 				// 1.3. Wrap paragraph's wrapper and sidenote in a section wrapper.
// 				//console.log("1.3. Wrap super wrappers")
// 				collectedElements.sidenote.addClass("temp-superwrapping-class")
// 				$(".temp-superwrapping-class").wrapAll("<section class=\"text-block\"></section>")
// 				$(".temp-superwrapping-class").removeClass("temp-superwrapping-class")
// 
// 				// 2. Put lastparagraph and nextsidenote into a new collectedElements group
// 				collectedElements.state = "collecting"
// 				collectedElements.paragraphs = [lastparagraph]
// 				collectedElements.sidenote = nextsidenote
// 			}
// 		} else {
// 			console.error("Unsupported state. Should never happen." + collectedElements.state)
// 		}
// 	})
// 
// 	// wrap remaining items if in "collecting" state
// 	if (collectedElements.state == "collecting") {
// 		// 1. Wrap collected elements.
// 		// 1.1. Insert sidenote after the last paragraph.
// 		//console.log("1.1. Insert after the last paragraph")
// 		collectedElements.sidenote.insertAfter(collectedElements.paragraphs[collectedElements.paragraphs.length - 1])
// 
// 		// 1.2. Wrap all paragraphs.
// 		//console.log("1.2. Wrap all paragraphs")
// 		$.each(collectedElements.paragraphs, function(index, value) {
// 			value.addClass("temp-wrapping-class")
// 		});
// 
// 		$(".temp-wrapping-class").wrapAll("<div class=\"text-main temp-superwrapping-class\"></div>")
// 
// 		$.each(collectedElements.paragraphs, function(index, value) {
// 			value.removeClass("temp-wrapping-class")
// 		});
// 
// 		// 1.3. Wrap paragraph's wrapper and sidenote in a section wrapper.
// 		//console.log("1.3. Wrap super wrappers")
// 		collectedElements.sidenote.addClass("temp-superwrapping-class")
// 		$(".temp-superwrapping-class").wrapAll("<section class=\"text-block\"></section>")
// 		$(".temp-superwrapping-class").removeClass("temp-superwrapping-class")
// 	}
// }
