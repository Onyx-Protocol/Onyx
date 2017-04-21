import Cocoa
import ServiceManagement
import Sparkle

class AppDelegate: NSObject, NSApplicationDelegate, SUUpdaterDelegate {

    var dashboardWindowController: DashboardWindowController?

    @IBAction func openDashboard(_ sender: AnyObject?) {
        if dashboardWindowController == nil {
            NSApp.keyWindow?.close()
            dashboardWindowController = NSStoryboard(name: "Main", bundle: nil).instantiateController(withIdentifier: "dashboard") as? DashboardWindowController
        }

        dashboardWindowController?.showWindow(nil)
    }

    @IBAction func showLicenses(_ sender: AnyObject?) {
        NSWorkspace.shared().open(URL(string: "https://chain.com/docs/core/reference/license")!)
    }

    @IBAction func showLogs(_ sender: AnyObject?) {
        NSWorkspace.shared().activateFileViewerSelecting([ChainCore.shared.logURL])
    }

    @IBAction func quitApp(_ sender: AnyObject?) {
        NSApp.terminate(nil)
    }

    @IBAction func openInTerminal(_ sender: AnyObject?) {
        openInTerminalApp("Terminal")
    }

    @IBAction func openInITerm(_ sender: AnyObject?) {
        openInTerminalApp("iTerm")
    }

    @IBAction func resetDatabase(_ sender: AnyObject?) {
        openDashboard(nil)
        DispatchQueue.main.async {
            let alert = NSAlert()
            alert.messageText = NSLocalizedString("Do you wish to reset the database? All data will be lost.", comment:"")
            alert.addButton(withTitle: "Keep Database")
            alert.addButton(withTitle: "Reset Database")
            alert.beginSheetModal(for: self.dashboardWindowController!.window!) { response in
                if response == NSAlertSecondButtonReturn {
                    NSLog("Resetting database.")
                    self.dashboardWindowController?.viewController.unloadDashboard()
                    ServerManager.shared.reset()
                    self.launchChainCore()
                }
            }
        }
    }

    @IBAction func showHelp(_ sender: AnyObject?) {
        NSWorkspace.shared().open(ChainCore.shared.docsURL)
    }

    func openInTerminalApp(_ appname: String) {
        let routine = "open_\(appname)"
        let param = String(format: "alias corectl='%@'; export DATABASE_URL='%@';",
                           arguments: [
                            ChainCore.shared.corectlPath.replacingOccurrences(of: " ", with: "\\ "),
                            ChainCore.shared.databaseURL
                            ])

        let launcher = ClientLauncher()
        do {
            try launcher.runSubroutine(routine, parameters: [param])
        } catch let error {
            if NSApp.keyWindow == nil {
                openDashboard(nil)
            }
            NSAlert(error: error).beginSheetModal(for: NSApp.keyWindow!, completionHandler: { (response) in
            })
        }
    }

    func launchChainCore() {
        dashboardWindowController!.viewController.beginAnimatingProgress()
        ServerManager.shared.launchIfNeeded()
    }

	func applicationWillFinishLaunching(_ notification: Notification) {
	}

	func applicationDidFinishLaunching(_ notification: Notification) {

        openDashboard(nil)

        DispatchQueue.main.async {
            if ServerManager.shared.failedTwice() {
                let alert = NSAlert()
                alert.messageText = NSLocalizedString("Chain Core may have been damaged. Do you wish to reset the database? All data will be lost.", comment:"")
                alert.addButton(withTitle: "Keep Database")
                alert.addButton(withTitle: "Reset Database")
                alert.beginSheetModal(for: self.dashboardWindowController!.window!) { response in
                    if response == NSAlertSecondButtonReturn {
                        NSLog("Damage detected, erasing database at user's request.")
                        ServerManager.shared.reset()
                        self.launchChainCore()
                    }
                }
            } else {
                self.launchChainCore()
            }
        }

        NSApp.activate(ignoringOtherApps: true)
	}

    func applicationWillTerminate(_ notification: Notification) {
        ServerManager.shared.terminate()
    }

    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        dashboardWindowController?.showWindow(nil)
        return true
    }
	
	
	func applicationDidBecomeActive(_ notification: Notification) {
        dashboardWindowController?.showWindow(nil)
	}
	

	// SUUpdater delegate methods
	func updater(_ updater: SUUpdater!, willInstallUpdate item: SUAppcastItem!) {
		print("updaterWillInstallUpdate")
	}
	
}

