$(function () {
	document.title = $("h1").text()

	prepareSidebarMenu()
	attachSignupFormToDownloadButton()
	selectOSForDownload()
	fixupSidenotes()
	prepareUpNextButton()
	loadVersionOptions()
})
