import Cocoa

class ServerManager: NSObject {
	
	static let shared = ServerManager()
    static let statusChangedNotification = NSNotification.Name("ServerManager.StatusChangedNotification")

    public var active:Bool = false
    public var ready:Bool = false
    public var error:NSError? = nil

    private var postgres:PostgresServer
    private var chainCore:ChainCore

    override init() {
        postgres = PostgresServer.builtin
        chainCore = ChainCore.shared
        super.init()
    }

    func launchIfNeeded() {
        active = true
        NotificationCenter.default.post(name: ServerManager.statusChangedNotification, object: self)

        // 1. Make sure the postgres server is launched
        // 2. Launch Chain Core
        postgres.start { pgStatus in
            switch pgStatus {
            case .Success:
                NSLog("ServerManager: Postgres is running.")
                self.chainCore.startIfNeeded { coreStatus in
                    switch coreStatus {
                    case .Success:
                        self.ready = true
                        self.error = nil
                        NSLog("ServerManager: Chain Core is running.")
                        self.clearFailures()
                    case .Failure(_):
                        // Failed to launch - retry.
                        self.chainCore.startIfNeeded { coreStatus in
                            switch coreStatus {
                            case .Success:
                                self.ready = true
                                self.error = nil
                                NSLog("ServerManager: Chain Core is running.")
                            case .Failure(let error):
                                self.ready = false
                                self.error = error
                                NSLog("ServerManager: Failed to launch Chain Core: %@", error.localizedDescription)
                            }
                            NotificationCenter.default.post(name: ServerManager.statusChangedNotification, object: self)
                        }
                    }
                    NotificationCenter.default.post(name: ServerManager.statusChangedNotification, object: self)
                }
            case .Failure(let error):
                NSLog("ServerManager: Failed to launch Postgres: %@", error.localizedDescription)
                self.ready = false
                self.error = error
                NotificationCenter.default.post(name: ServerManager.statusChangedNotification, object: self)
            }
        }
    }

    func terminate() {
        NSLog("ServerManager: Stopping Chain Core.")
        chainCore.stopSync()
        NSLog("ServerManager: Stopping Postgres.")
        postgres.stopSync()
        active = false
        ready = false
    }

    func reset() {
        self.terminate()
        sleep(1)
        postgres.reset()
        chainCore.reset()
    }

    private var registeringError: Bool = false
    func registerFailure(_ message: String) {
        if registeringError {
            return
        }
        registeringError = true
        // Increment number of failed attempts.
        writeFailureCount(readFailureCount() + 1)

        let alert = NSAlert()
        alert.messageText = message
        alert.addButton(withTitle: "Quit")
        if let win = NSApp.keyWindow {
            alert.beginSheetModal(for: win) { response in
                NSApp.terminate(nil)
            }
        } else {
            alert.runModal()
            NSApp.terminate(nil)
        }
    }

    func clearFailures() {
        writeFailureCount(0)
    }

    func failedTwice() -> Bool {
        return readFailureCount() >= 2
    }

    private func readFailureCount() -> Int {
        if let data = NSData(contentsOf: failureCountFileURL()) {
            return ((String(data: data as Data, encoding: .utf8) ?? "0") as NSString).integerValue
        } else {
            return 0
        }
    }

    private func writeFailureCount(_ count: Int) {
        let url = failureCountFileURL()
        if count == 0 {
            try? FileManager.default.removeItem(at: url)
            return
        }
        do {
            try "\(count)".data(using: .utf8)?.write(to: url, options: .atomic)
        } catch let error {
            NSLog("ServerManager: Failed to write the failure counter to %@: %@", url.absoluteString, error.localizedDescription)
        }
    }

    private func failureCountFileURL() -> URL {
        return URL(fileURLWithPath: FileManager().applicationSupportDirectoryPath().appendingFormat("/failures.txt"))
    }
}


