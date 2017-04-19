import Cocoa

class DashboardWindowController: NSWindowController, NSWindowDelegate {

    var observer: NSObjectProtocol?

    var viewController: DashboardViewController {
        return self.contentViewController as! DashboardViewController
    }

    var showingError:Bool = false

    override func awakeFromNib() {
        super.awakeFromNib()
        self.windowFrameAutosaveName = "DashboardWindowPosition_16_20"; // the setting in IB does not help
    }

    override func windowDidLoad() {
        super.windowDidLoad()

        if UserDefaults.standard.object(forKey: "NSWindow Frame \(self.windowFrameAutosaveName)") == nil {
            let screenFrame = NSScreen.main()!.frame
            var windowFrame = self.window!.frame

            // make window frame size proportional to the screen:
            let sizeRatio:CGFloat = 0.66
            windowFrame.size.width = min(max(1150, round(sizeRatio*screenFrame.size.width)), screenFrame.size.width)
            windowFrame.size.height = round(1.1*sizeRatio*screenFrame.size.height) // screens nowadays are too narrow vertically - make the window a bit higher

            let topToBottomRatio:CGFloat = 0.55 // per HIG we want top space be 50% of the space below window

            // H = bottom*ratio + h + bottom
            // bottom = (H - h)/(ratio+1)
            windowFrame.origin.x = (screenFrame.size.width - windowFrame.size.width)/2
            windowFrame.origin.y = (screenFrame.size.height - windowFrame.size.height)/(1.0 + topToBottomRatio)

            self.window?.setFrame(windowFrame, display: true)
        }

        if #available(OSX 10.12, *) {
            NSWindow.allowsAutomaticWindowTabbing = false
        }

        viewController.statusLabel.stringValue = "Version \(Bundle.main.infoDictionary![kCFBundleVersionKey as String] ?? "")"

        NotificationCenter.default.addObserver(forName: ServerManager.statusChangedNotification, object: nil, queue: OperationQueue.main) { _ in

            if ServerManager.shared.ready {
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.066) {
                    self.viewController.loadDashboard()
                }

            } else if let error = ServerManager.shared.error {
                self.viewController.statusLabel.stringValue = error.localizedDescription

                if !self.showingError {
                    self.showingError = true
                    NSAlert(error: error).beginSheetModal(for: self.window!, completionHandler: { (response) in
                        self.showingError = false
                        NSApp.terminate(nil)
                    })
                }
            }
        }
    }
    
    func windowWillClose(_ notification: Notification) {
        if let o = observer {
            NotificationCenter.default.removeObserver(o)
            observer = nil
        }
    }
}
