import Cocoa
import WebKit

class WebContentWindowController: NSWindowController, NSWindowDelegate {

    var title: String? {
        didSet {
            self.window?.title = title ?? ""
        }
    }
    var url: URL? {
        didSet {
            if let u = url {
                if self.window != nil {
                    self.viewController.doLoadWebView(url: u)
                }
            }
        }
    }

    var viewController: WebContentViewController {
        return self.contentViewController as! WebContentViewController
    }

    var showingError:Bool = false

    override func awakeFromNib() {
        super.awakeFromNib()
        self.windowFrameAutosaveName = "WebViewWindowPosition_16_20"; // the setting in IB does not help
    }

    override func windowDidLoad() {
        super.windowDidLoad()

        if UserDefaults.standard.object(forKey: "NSWindow Frame \(self.windowFrameAutosaveName ?? "n/a")") == nil {
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
            NSWindow.allowsAutomaticWindowTabbing = true
        }

        if let url = self.url {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.066) {
                self.viewController.doLoadWebView(url: url)
            }
        }
    }

    func windowWillClose(_ notification: Notification) {
        AppDelegate.shared.closeWebContent(title: title ?? "")
    }
}





class WebContentViewController: NSViewController, WebUIDelegate, WKUIDelegate, WKNavigationDelegate, WKScriptMessageHandler {

    @IBOutlet weak var webViewOld: WebView!
    @IBOutlet weak var webView: WKWebView!

    override func viewDidLoad() {
        super.viewDidLoad()
    }

    func userAgent() -> String {
        return "ChainCore.app/\(Bundle.main.infoDictionary![kCFBundleVersionKey as String] ?? "")"
    }

    func doLoadModernWebView(url: URL) {
        if #available(OSX 10.10, *) {
            if webView != nil {
                return
            }
            let config = WKWebViewConfiguration()
            if #available(OSX 10.11, *) {
                config.websiteDataStore = WKWebsiteDataStore.default()
                config.applicationNameForUserAgent = userAgent()
            }
            let ctrl = WKUserContentController()

            let consoleOverride = "window.console = { }"
            let userScript = WKUserScript(source: consoleOverride, injectionTime: WKUserScriptInjectionTime.atDocumentStart, forMainFrameOnly: true)
            ctrl.addUserScript(userScript)

            for name in ["log", "warn", "error", "debug", "info"] {
                ctrl.add(self, name: name)
                // Override console.log so we intercept it.
                let methodOverride = "window.console.\(name) = function(msg) { window.webkit.messageHandlers.\(name).postMessage(msg); };"
                let userScript = WKUserScript(source: methodOverride, injectionTime: WKUserScriptInjectionTime.atDocumentStart, forMainFrameOnly: true)
                ctrl.addUserScript(userScript)
            }

            config.userContentController = ctrl
            let wv = WKWebView(frame: self.view.bounds, configuration: config)
            wv.autoresizingMask = [.viewWidthSizable, .viewHeightSizable]
            wv.translatesAutoresizingMaskIntoConstraints = true

            self.view.addSubview(wv)
            wv.uiDelegate = self

            DispatchQueue.main.asyncAfter(deadline: .now() + 0.2, execute: {
                wv.load(URLRequest(url: url))
            })

            //            // Debug:
            //            DispatchQueue.main.asyncAfter(deadline: .now() + 3, execute: {
            //                wv.evaluateJavaScript("console.error('test from js bridge')", completionHandler: { (result, err) in
            //                    //                    NSLog("Executed JS: %@ %@", "\(result)", "\(err)")
            //                })
            //            })

            webView = wv
        }
    }

    func doLoadLegacyWebView(url: URL) {
        if webViewOld != nil {
            return
        }
        let wv = WebView(frame: self.view.bounds)
        wv.autoresizingMask = [.viewWidthSizable, .viewHeightSizable]
        wv.translatesAutoresizingMaskIntoConstraints = true
        self.view.addSubview(wv)

        wv.uiDelegate = self
        wv.customUserAgent = userAgent()

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.2, execute: {
            wv.mainFrame.load(URLRequest(url: url))
        })

        webViewOld = wv
    }

    func doLoadWebView(url: URL) {
        if #available(OSX 10.10, *) {
            doLoadModernWebView(url: url)
        } else {
            doLoadLegacyWebView(url: url)
        }
    }

    // New webview delegate

    func webView(_ webView: WKWebView, runJavaScriptAlertPanelWithMessage message: String, initiatedByFrame frame: WKFrameInfo, completionHandler: @escaping () -> Void) {
        let alert = NSAlert()
        alert.addButton(withTitle: "OK")
        alert.messageText = message
        alert.beginSheetModal(for: self.view.window!) { response in
            completionHandler()
        }
    }

    func webView(_ webView: WKWebView, runJavaScriptConfirmPanelWithMessage message: String, initiatedByFrame frame: WKFrameInfo, completionHandler: @escaping (Bool) -> Void) {
        let alert = NSAlert()
        alert.messageText = message
        alert.addButton(withTitle: "OK")
        alert.addButton(withTitle: "Cancel")
        alert.beginSheetModal(for: self.view.window!) { response in
            if response == NSAlertFirstButtonReturn {
                completionHandler(true)
                return
            }
            completionHandler(false)
        }
    }

    func webView(_ webView: WKWebView, createWebViewWith configuration: WKWebViewConfiguration, for navigationAction: WKNavigationAction, windowFeatures: WKWindowFeatures) -> WKWebView? {
        if let url = navigationAction.request.url, url.absoluteString.contains("/dashboard") {
            AppDelegate.shared.dashboardWindowController?.window?.makeKeyAndOrderFront(nil)
            return nil
        }

        if !(navigationAction.targetFrame?.isMainFrame ?? false) {
            if let url = navigationAction.request.url {
                NSWorkspace.shared().open(url)
            }
        }
        return nil
    }

    func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        NSLog("Chain Core js.console.%@: %@", "\(message.name)","\(message.body)")
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        self.view.setNeedsDisplay(self.view.bounds)
        webView.setNeedsDisplay(webView.bounds)
    }


    // Old webview delegate

    func webView(_ sender: WebView!, runJavaScriptAlertPanelWithMessage message: String!, initiatedBy frame: WebFrame!) {
        let alert = NSAlert()
        alert.addButton(withTitle: "OK")
        alert.messageText = message
        alert.runModal()
    }

    func webView(_ sender: WebView!, runJavaScriptConfirmPanelWithMessage message: String!, initiatedBy frame: WebFrame!) -> Bool {
        let alert = NSAlert()
        alert.messageText = message
        alert.addButton(withTitle: "OK")
        alert.addButton(withTitle: "Cancel")
        if alert.runModal() == NSAlertFirstButtonReturn {
            return true
        }
        return false
    }


    // Actions

    @IBAction func reloadPage(_ sender: Any?) {
        webView?.reload(sender)
        webViewOld?.reload(sender)
    }
    
    @IBAction func goBack(_ sender: Any?) {
        webView?.goBack(sender)
        webViewOld?.goBack(sender)
    }
    
    @IBAction func goForward(_ sender: Any?) {
        webView?.goForward(sender)
        webViewOld?.goForward(sender)
    }
    
}

