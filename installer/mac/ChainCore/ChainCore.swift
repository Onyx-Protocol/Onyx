import Cocoa

// Manages the cored process
class ChainCore: NSObject {

    let port: UInt
    let dbPort: UInt

    private var queue: DispatchQueue
    private var task: Process?
    private var taskCleaner: TaskCleaner?
    private var expectingTermination: Bool = false

    var databaseURL: String {
        return "postgres://localhost:\(dbPort)/core?sslmode=disable"
    }

    var corectlPath: String {
        return Bundle.main.path(forResource: "corectl", ofType: nil)!
    }

    var dashboardURL: URL {
        return URL(string: "http://localhost:\(port)/dashboard")!
    }

    var infoURL: URL {
        return URL(string: "http://localhost:\(port)/info")!
    }

    var docsURL: URL {
        return URL(string: "http://localhost:\(port)/docs")!
    }

    var logURL: URL {
        return URL(fileURLWithPath: FileManager().applicationSupportDirectoryPath().appendingFormat("/cored.log"))
    }

    static var shared: ChainCore = ChainCore()

    override init() {
        self.port = Config.shared.corePort
        self.dbPort = Config.shared.dbPort
        self.queue = DispatchQueue(label: "com.chain.cored")
        super.init()
    }

    func reset() {
    }

    // Main thread
    func startIfNeeded(_ completion: @escaping (OperationStatus) -> Void) {
        expectingTermination = false
        if let t = self.task {
            if t.isRunning {
                completion(.Success)
                return
            } else {
                self.task = nil
            }
        }

        start(completion)
    }

    func start(_ completion: @escaping (OperationStatus) -> Void) {

        if Config.portInUse(self.port) {
            completion(.Failure(NSError(domain: "com.chain.ChainCore.cored-status", code: 0,
                userInfo: [NSLocalizedDescriptionKey: "Chain Core cannot run on localhost:\(self.port), port is in use."])))
            return
        }

        self.task?.terminate()
        let t = makeTask()
        self.task = t
        queue.async {
            t.launch()
            let pid = t.processIdentifier
            DispatchQueue.main.async {
                self.taskCleaner?.terminate()
                self.taskCleaner = TaskCleaner(childPid: pid)
                self.taskCleaner?.watch()
            }
            t.waitUntilExit()
            if !self.expectingTermination {
                // If task died unexpectedly, kill the whole app. We are not smart enough to deal with restarts while our tasks are robust enough not to die spontaneously.
                NSLog("Chain Core stopped unexpectedly. Exiting.")
                DispatchQueue.main.async {
                    ServerManager.shared.registerFailure("Chain Core stopped unexpectedly. Please check the logs and try again.")
                }
            }
        }
        DispatchQueue.global().asyncAfter(deadline: .now() + 1.0) {
            let testRequest = URLSession.shared.dataTask(with: self.infoURL, completionHandler: { (data, response, error) in
                DispatchQueue.main.async {
                    if let t = self.task {
                        if t.isRunning {
                            if data != nil {
                                DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
                                    completion(.Success)
                                }
                            } else if error != nil {
                                self.task?.terminate()
                                self.task = nil
                                completion(.Failure(error! as NSError))
                            } else {
                                self.task?.terminate()
                                self.task = nil
                                completion(.Failure(NSError(domain: "com.chain.ChainCore.cored-status", code: 0, userInfo: [NSLocalizedDescriptionKey: "Chain Core failed to respond"])))
                            }
                        } else {
                            self.task = nil
                            completion(.Failure(NSError(domain: "com.chain.ChainCore.cored-status", code: 0, userInfo: [NSLocalizedDescriptionKey: "Chain Core failed to launch"])))
                        }
                    } else {
                        completion(.Failure(NSError(domain: "com.chain.ChainCore.cored-status", code: 0, userInfo: [NSLocalizedDescriptionKey: "Chain Core was stopped"])))
                    }
                }
            })
            testRequest.resume()
        }

    }

    func makeTask() -> Process {
        let task = Process()
        task.launchPath = Bundle.main.path(forResource: "cored", ofType: nil)
        task.arguments = []
        task.environment = [
            "DATABASE_URL": databaseURL,
            "LISTEN":       ":\(port)",
            "LOGFILE":      self.logURL.path,

            // FIXME: cored binaries built with bin/build-cored-release have trouble acquiring a default user for Postgres connections. This ensures the current user's login name is always available in the environment.
            "USER":         NSUserName(),
        ]
        //task.standardOutput = Pipe()
        //task.standardError = Pipe()
        return task
    }

    func stopSync() {
        expectingTermination = true
        self.taskCleaner?.terminate()
        self.taskCleaner = nil
        self.task?.terminate()
        self.task = nil
    }
}
