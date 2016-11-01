
# ATTENTION: if this fails to compile with OSACompile, install iTerm in /Applications.
# You do not need to run it. Just put it there, so AppleScript knows that it is defined.
# You may also try to rebuild database:
# $ /System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister -kill -r -domain local -domain system -domain user

on open_Terminal(cmnd)
	tell application "Terminal"
		activate
		if number of windows = 0 then
			do script cmnd
        else
			tell application "System Events" to keystroke "t" using command down
            delay 0.5
			do script cmnd in window 1
		end if
	end tell
end open_Terminal


on open_iTerm(cmnd)
	tell application "iTerm"
		activate
		if number of windows = 0 then
			create window with default profile command cmnd
			else
			create tab with default profile command cmnd
		end if
	end tell
end openITerm

