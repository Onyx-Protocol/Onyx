import Cocoa
import WebKit


class DashboardViewController: NSViewController, WebUIDelegate, WKUIDelegate, WKNavigationDelegate, WKScriptMessageHandler {

    @IBOutlet weak var preloadView: NSView!
    @IBOutlet weak var iconView: NSImageView!
    @IBOutlet weak var titleLabel: NSTextField!
    @IBOutlet weak var subtitleLabel: NSTextField!
    @IBOutlet weak var statusLabel: NSTextField!

    @IBOutlet weak var webViewOld: WebView!
    @IBOutlet weak var webView: WKWebView!

    @IBOutlet weak var progressBarConstraint: NSLayoutConstraint!
    @IBOutlet weak var progressTrackView: NSBox!

    func beginAnimatingProgress() {
        progressBarConstraint.constant = 1

        NSAnimationContext.runAnimationGroup({ (ctx) in
            ctx.allowsImplicitAnimation = true
            ctx.duration = 4.50 // this is an estimated duration.
            ctx.timingFunction = CAMediaTimingFunction(name: kCAMediaTimingFunctionEaseInEaseOut)
            progressBarConstraint.animator().constant = 0.8 * progressTrackView.frame.size.width
        }, completionHandler: {

        })
    }

    override func viewDidLoad() {
        super.viewDidLoad()

        progressBarConstraint.constant = 1.0

        self.subtitleLabel.font = NSFont(name: "Nitti-Bold", size: 22)
        self.titleLabel.font    = NSFont(name: "NittiGrotesk-Bold", size: 60)
        self.statusLabel.font   = NSFont(name: "NittiGrotesk-Medium", size: 16)

        self.subtitleLabel.attributedStringValue = NSAttributedString(string: self.subtitleLabel.stringValue, attributes: [
            NSKernAttributeName: 0.6
        ])
        self.titleLabel.attributedStringValue = NSAttributedString(string: self.titleLabel.stringValue, attributes: [
            NSKernAttributeName: 1.5
        ])
    }

    func showLicense() {

    }

    func unloadDashboard() {
        preloadView.isHidden = false
        progressBarConstraint.constant = 1.0

        webView?.navigationDelegate = nil
        webView?.uiDelegate = nil
        webView?.removeFromSuperview()
        webView = nil

        webViewOld?.uiDelegate = nil
        webViewOld?.removeFromSuperview()
        webViewOld = nil
    }

    func loadDashboard() {

        // Make sure progress bar animation finishes smoothly.
        NSAnimationContext.runAnimationGroup({ (ctx) in
            ctx.allowsImplicitAnimation = true
            ctx.duration = 0.25
            ctx.timingFunction = CAMediaTimingFunction(name: kCAMediaTimingFunctionEaseOut)
            progressBarConstraint.animator().constant = progressTrackView.frame.size.width
        }, completionHandler: {
            self.doLoadDashboard()
        })
    }

    func doLoadDashboard() {
        if #available(OSX 10.11, *) {
            if webView != nil {
                return
            }
            let config = WKWebViewConfiguration()
            config.websiteDataStore = WKWebsiteDataStore.default()
            config.applicationNameForUserAgent = "ChainCore.app/\(Bundle.main.infoDictionary![kCFBundleVersionKey as String])"
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
            self.preloadView.isHidden = true
            wv.uiDelegate = self
            wv.load(URLRequest(url: ChainCore.shared.dashboardURL))

//            // Debug:
//            DispatchQueue.main.asyncAfter(deadline: .now() + 3, execute: { 
//                wv.evaluateJavaScript("console.error('test from js bridge')", completionHandler: { (result, err) in
//                    //                    NSLog("Executed JS: %@ %@", "\(result)", "\(err)")
//                })
//            })

            webView = wv
        } else {
            if webViewOld != nil {
                return
            }
            let wv = WebView(frame: self.view.bounds)
            wv.autoresizingMask = [.viewWidthSizable, .viewHeightSizable]
            wv.translatesAutoresizingMaskIntoConstraints = true
            self.view.addSubview(wv)
            self.preloadView.isHidden = true

            wv.mainFrame.load(URLRequest(url: ChainCore.shared.dashboardURL))
            wv.uiDelegate = self
            wv.customUserAgent = "ChainCore.app/\(Bundle.main.infoDictionary![kCFBundleVersionKey as String])"

            webViewOld = wv
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
        if !(navigationAction.targetFrame?.isMainFrame ?? false) {
            if let url = navigationAction.request.url {
                NSWorkspace.shared().open(url)
            }
        }
        return nil
    }

    public func userContentController(_ userContentController: WKUserContentController, didReceive message: WKScriptMessage) {
        NSLog("Chain Core js.console.%@: %@", "\(message.name)","\(message.body)")
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

