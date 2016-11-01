import Cocoa

class PostgresServer: NSObject {

    static let BuiltinVersion = "9.5"
    static var builtin = PostgresServer("Builtin Postgres")

    //static let PostgresPath = "/Applications/ChainCore.app/Contents/Postgres"
    static let PostgresPath = "\(Bundle.main.bundlePath)/Contents/Postgres"
	
	static let PropertyChangedNotification = NSNotification.Name("PostgresServer.PropertyChangedNotification")
	static let StatusChangedNotification = NSNotification.Name("PostgresServer.StatusChangedNotification")
	

	@objc enum ServerStatus: Int {
		case NoBinaries
		case PortInUse
		case DataDirInUse
		case DataDirIncompatible
		case DataDirEmpty
		case Running
		case Startable
		case StalePidFile
		case Unknown
	}

	dynamic var name: String = "" {
		didSet {
			NotificationCenter.default.post(name: PostgresServer.PropertyChangedNotification, object: self)
		}
	}
	dynamic var version: String = ""
	dynamic var port: UInt = 0 {
		didSet {
			NotificationCenter.default.post(name: PostgresServer.PropertyChangedNotification, object: self)
		}
	}
	dynamic var binPath: String = ""
	dynamic var varPath: String = ""
	dynamic var startAtLogin: Bool = false {
		didSet {
			NotificationCenter.default.post(name: PostgresServer.PropertyChangedNotification, object: self)
		}
	}
	dynamic var configFilePath: String {
		return varPath.appending("/postgresql.conf")
	}
	dynamic var hbaFilePath: String {
		return varPath.appending("/pg_hba.conf")
	}
	dynamic var logFilePath: String {
		return varPath.appending("/postgresql.log")
	}
	private var pidFilePath: String {
		return varPath.appending("/postmaster.pid")
	}
	private var pgVersionPath: String {
		return varPath.appending("/PG_VERSION")
	}

    private var queue = DispatchQueue(label: "com.chain.cored")
    private var task: Process?
    private var taskCleaner: TaskCleaner?
    private var expectingTermination:Bool = false

	dynamic private(set) var running: Bool = false
	dynamic private(set) var serverStatus: ServerStatus = .Unknown
    dynamic private(set) var statusMessage: String = "" {
        didSet {
            NSLog("DEBUG: PostgresServer.statusMessage = %@", "\(statusMessage)")
        }
    }
	dynamic private(set) var databases: [Database] = []

	convenience init(_ name: String, _ version: String? = nil, _ port: UInt = Config.shared.dbPort, _ varPath: String? = nil) {
		self.init()
		
		self.name = name
		self.version = PostgresServer.BuiltinVersion
		self.port = port
		self.binPath = PostgresServer.PostgresPath.appending("/bin")
        let appVersion:String = (Bundle.main.infoDictionary!["ChainDatabaseVersion"] as? String) ?? "default"
		self.varPath = varPath ?? FileManager().applicationSupportDirectoryPath().appendingFormat("/postgres-%@-db-%@", self.version, appVersion)

//        NSLog("PostgresServer: binPath = %@", self.binPath)
//        NSLog("PostgresServer: varPath = %@", self.varPath)



		updateServerStatus()
		
		// TODO: read port from postgresql.conf
	}

    func reset() {
        try? FileManager.default.removeItem(at: URL(fileURLWithPath: varPath))
    }

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


	// MARK: Async handlers
	func start(_ completion: @escaping (OperationStatus) -> Void) {
		updateServerStatus()

		DispatchQueue.global().async {
			let statusResult: OperationStatus

			switch self.serverStatus {
			
			case .NoBinaries:
				let userInfo = [
					NSLocalizedDescriptionKey: NSLocalizedString("The binaries for this PostgreSQL server were not found", comment: ""),
					NSLocalizedRecoverySuggestionErrorKey: "Create a new Server and try again."
				]
				statusResult = .Failure(NSError(domain: "com.chain.ChainCore.server-status", code: 0, userInfo: userInfo))
				NSLog("DEBUG: PostgresServer.start - no binaries")
			case .PortInUse:
				let userInfo = [
					NSLocalizedDescriptionKey: NSLocalizedString("Port \(self.port) is already in use", comment: ""),
					NSLocalizedRecoverySuggestionErrorKey: "Usually this means that there is already a PostgreSQL server running on your Mac. If you want to run multiple servers simultaneously, use different ports."
				]
				statusResult = .Failure(NSError(domain: "com.chain.ChainCore.server-status", code: 0, userInfo: userInfo))
				NSLog("DEBUG: PostgresServer.start - port in use")
			case .DataDirInUse:
				let userInfo = [
					NSLocalizedDescriptionKey: NSLocalizedString("There is already a PostgreSQL server running in this data directory", comment: ""),
				]
				statusResult = .Failure(NSError(domain: "com.chain.ChainCore.server-status", code: 0, userInfo: userInfo))
				NSLog("DEBUG: PostgresServer.start - data dir in use")
			case .DataDirIncompatible:
				let userInfo = [
					NSLocalizedDescriptionKey: NSLocalizedString("The data directory is not compatible with this version of PostgreSQL server.", comment: ""),
					NSLocalizedRecoverySuggestionErrorKey: "Please create a new Server."
				]
				statusResult = .Failure(NSError(domain: "com.chain.ChainCore.server-status", code: 0, userInfo: userInfo))
				NSLog("DEBUG: PostgresServer.start - data dir incompatible")
			case .DataDirEmpty:
                //NSLog("DEBUG: PostgresServer.start - data dir is empty. Starting initdb...")
                let initResult = self.initDatabaseSync()

                //NSLog("DEBUG: PostgresServer.start - initdb finished.")
				if case .Failure = initResult {
                    NSLog("DEBUG: PostgresServer.start - initdb failed, actually.")
					statusResult = initResult
					break
				}

                //NSLog("DEBUG: PostgresServer.start - doStart...")
                self.doStart()
                sleep(1) // let postgres actually launch

                //NSLog("DEBUG: PostgresServer.start - doStart warmed up, creating user...")
				let createUserResult = self.createUserSync()
				guard case .Success = createUserResult else {
                    NSLog("DEBUG: PostgresServer.start - failed to create user.")
					statusResult = createUserResult
					break
				}

                //NSLog("DEBUG: PostgresServer.start - doStart warmed up, creating user database '%@'...", "\(NSUserName())")
				
                let createDBResult = self.createDatabaseSync(name: NSUserName())
				if case .Failure = createDBResult {
                    NSLog("DEBUG: PostgresServer.start - failed to create user db.")
					statusResult = createDBResult
					break
				}

                //NSLog("DEBUG: PostgresServer.start - creating database 'core'...")
                let createChainCoreDBResult = self.createDatabaseSync(name: "core")
                if case .Failure = createChainCoreDBResult {
                    NSLog("DEBUG: PostgresServer.start - failed to create core db.")
                    statusResult = createChainCoreDBResult
                    break
                }
				
				statusResult = .Success
				
			case .Running:
                NSLog("DEBUG: PostgresServer.start - already running.")
				statusResult = .Success
				
			case .Startable:
                //NSLog("DEBUG: PostgresServer.start: startable - starting...")
                self.doStart()
                sleep(1)
                NSLog("DEBUG: PostgresServer.start - doStart done")
                DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
                    NSLog("DEBUG: PostgresServer.start - reporting success.")
                    completion(.Success)
                }
                return

			case .StalePidFile:
				let userInfo = [
					NSLocalizedDescriptionKey: NSLocalizedString("The data directory contains an old postmaster.pid file", comment: ""),
					NSLocalizedRecoverySuggestionErrorKey: "The data directory contains a postmaster.pid file, which usually means that the server is already running. When the server crashes or is killed, you have to remove this file before you can restart the server. Make sure that the database process is definitely not runnnig anymore, otherwise your data directory will be corrupted."
				]
				statusResult = .Failure(NSError(domain: "com.chain.ChainCore.server-status", code: 0, userInfo: userInfo))
				NSLog("DEBUG: PostgresServer.start: stale pid file")
			case .Unknown:
				let userInfo = [
					NSLocalizedDescriptionKey: NSLocalizedString("Unknown server status", comment: ""),
					NSLocalizedRecoverySuggestionErrorKey: ""
				]
				statusResult = .Failure(NSError(domain: "com.chain.ChainCore.server-status", code: 0, userInfo: userInfo))
				NSLog("DEBUG: PostgresServer.start: unknown issue")
			}

			
			DispatchQueue.main.async {
				self.updateServerStatus()
				completion(statusResult)
			}
			
		}
	}
	

	/// Checks if the server is running.
	/// Must be called only from the main thread.
	func updateServerStatus() {
		if !FileManager.default.fileExists(atPath: binPath) {
			serverStatus = .NoBinaries
			running = false
			statusMessage = "No binaries found"
			databases.removeAll()
			return
		}
		
		if !FileManager.default.fileExists(atPath: pgVersionPath) {
			serverStatus = .DataDirEmpty
			running = false
			statusMessage = "Click ‘Start’ to initialize the server"
			databases.removeAll()
			return
		}
		
		do {
			let versionFileContent = try String(contentsOfFile: pgVersionPath)
			if version != versionFileContent.substring(to: versionFileContent.index(before: versionFileContent.endIndex)) {
				serverStatus = .DataDirIncompatible
				running = false
				statusMessage = "Database directory incompatible"
				databases.removeAll()
				return
			}
		} catch {
			serverStatus = .Unknown
			running = false
			statusMessage = "Could not determine data directory version"
			databases.removeAll()
			return
		}
		
		if FileManager.default.fileExists(atPath: pidFilePath) {
			guard let pidFileContents = try? String(contentsOfFile: pidFilePath, encoding: .utf8) else {
				serverStatus = .Unknown
				running = false
				statusMessage = "Could not read PID file"
				databases.removeAll()
				return
			}
			
			let firstLine = pidFileContents.components(separatedBy: .newlines).first!
			guard let pid = Int32(firstLine) else {
				serverStatus = .Unknown
				running = false
				statusMessage = "First line of PID file is not an integer"
				databases.removeAll()
				return
			}
			
			var buffer = [CChar](repeating: 0, count: 1024)
			proc_pidpath(pid, &buffer, UInt32(buffer.count))
			let processPath = String(cString: buffer)
			
			if processPath == binPath.appending("/postgres") {
				serverStatus = .Running
				running = true
				statusMessage = "PostgreSQL \(self.version) - Running on port \(self.port)"
				databases.removeAll()
                loadDatabases()
				return
			}
			else if processPath.hasSuffix("postgres") || processPath.hasSuffix("postmaster") {
				serverStatus = .DataDirInUse
				running = false
				statusMessage = "The data directory is in use by another server"
				databases.removeAll()
				return
			}
			else if !processPath.isEmpty {
				serverStatus = .StalePidFile
				running = false
				statusMessage = "Old postmaster.pid file detected"
				databases.removeAll()
				return
			}
		}
		
		if Config.portInUse(port) {
			serverStatus = .PortInUse
			running = false
			statusMessage = "Port in use by another process"
            NSLog("DEBUG: PostgresServer.updateServerStatus - port in use: %@", "\(port)")
			databases.removeAll()
			return
		}
		
		serverStatus = .Startable
		running = false
		statusMessage = "Not running"
		databases.removeAll()
	}

	
	/// Loads the databases from the servers.
	private func loadDatabases() {
		databases.removeAll()

        NSLog("DEBUG: PostgresServer.loadDatabases")
		
		let url = "postgresql://:\(port)"
		let connection = PQconnectdb(url.cString(using: .utf8))
		
		if PQstatus(connection) == CONNECTION_OK {
            NSLog("DEBUG: PostgresServer.loadDatabases - connection ok")
			let result = PQexec(connection, "SELECT datname FROM pg_database WHERE datallowconn ORDER BY LOWER(datname)")
			for i in 0..<PQntuples(result) {
				guard let value = PQgetvalue(result, i, 0) else { continue }
				let name = String(cString: value)
				databases.append(Database(name))
			}
			PQfinish(connection)
            NSLog("DEBUG: PostgresServer.loadDatabases - finished connection")

		}
	}

    func doStart() {
        self.task?.interrupt()
        let t = self.makeTask()
        self.task = t
        self.queue.async {
            t.launch()
            let pid = t.processIdentifier
            DispatchQueue.main.async {
                self.taskCleaner?.terminate()
                self.taskCleaner = TaskCleaner(childPid: pid)
                self.taskCleaner?.watch()
            }
            t.waitUntilExit()

            NSLog("DEBUG: PostgresServer.doStart: task stopped.")
            if !self.expectingTermination {
                // If task died unexpectedly, kill the whole app. We are not smart enough to deal with restarts while our tasks are robust enough not to die spontaneously.
                NSLog("Postgres stopped unexpectedly. Exiting.")
                DispatchQueue.main.async {
                    ServerManager.shared.registerFailure("Database stopped unexpectedly. Please check the logs and try again.")
                }
            }
        }
    }

    func makeTask() -> Process {
        let task = Process()
        task.launchPath = binPath.appending("/postgres")
        let libpath = "\(PostgresServer.PostgresPath)/lib"
        task.environment = [
            "DYLD_LIBRARY_PATH": libpath,
//            "DYLD_FALLBACK_LIBRARY_PATH": libpath,
//            "DYLD_VERSIONED_LIBRARY_PATH": libpath
        ]
        task.arguments = [
            "-D", varPath,
            "-c", "port=\(port)",
            "-c", "logging_collector=true",
        ]
        //task.standardOutput = Pipe()
        //task.standardError = Pipe()
        return task
    }

	
	func stopSync() {
        expectingTermination = true
        self.taskCleaner?.terminate()
        self.taskCleaner = nil
        self.task?.interrupt()
        self.task = nil
	}
	
	
	private func initDatabaseSync() -> OperationStatus {

		let task = Process()
		task.launchPath = binPath.appending("/initdb")
        let libpath = "\(PostgresServer.PostgresPath)/lib"
        //NSLog("libpath = %@", libpath)
        task.environment = [
            "COREAPP_DYLD_LIBRARY_PATH": libpath,
            "PGHOST": "localhost"
        ]
		task.arguments = [
			"-D", varPath,
			"-U", "postgres",
			"--encoding=UTF-8",
			"--locale=en_US.UTF-8"
		]

        // NOTE: on Devon's machine on Oct 21, 2016 we had mysterious hang up on initdb if we used these null pipes.
        // Without them it did not hang up.
//		task.standardOutput = Pipe()
//		let errorPipe = Pipe()
//		task.standardError = errorPipe
		task.launch()
        let errorDescription = "" //String(data: errorPipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? "(incorrectly encoded error message)"
		task.waitUntilExit()
		
		if task.terminationStatus == 0 {
			return .Success
		} else {

            NSLog("DEBUG: failed to launch initdb")
			let userInfo: [String: Any] = [
				NSLocalizedDescriptionKey: NSLocalizedString("Could not initialize database cluster.", comment: ""),
				NSLocalizedRecoverySuggestionErrorKey: errorDescription,
				NSLocalizedRecoveryOptionsErrorKey: ["OK", "Open Server Log"],
				NSRecoveryAttempterErrorKey: ErrorRecoveryAttempter(recoveryAttempter: { (error, optionIndex) -> Bool in
					if optionIndex == 1 {
						NSWorkspace.shared().openFile(self.logFilePath, withApplication: "Console")
					}
					return true
				})
			]
			return .Failure(NSError(domain: "com.chain.ChainCore.initdb", code: 0, userInfo: userInfo))
		}
	}
	
	
	private func createUserSync() -> OperationStatus {
		let task = Process()
		task.launchPath = binPath.appending("/createuser")
        task.environment = [
            "PGHOST": "localhost"
        ]
		task.arguments = [
			"-U", "postgres",
			"-p", String(port),
			"--superuser",
			NSUserName()
		]
		task.standardOutput = Pipe()
		let errorPipe = Pipe()
		task.standardError = errorPipe
		task.launch()
		let errorDescription = String(data: errorPipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? "(incorrectly encoded error message)"
		task.waitUntilExit()
		
		if task.terminationStatus == 0 {
			return .Success
		} else {
			let userInfo: [String: Any] = [
				NSLocalizedDescriptionKey: NSLocalizedString("Could not create default user.", comment: ""),
				NSLocalizedRecoverySuggestionErrorKey: errorDescription,
				NSLocalizedRecoveryOptionsErrorKey: ["OK", "Open Server Log"],
				NSRecoveryAttempterErrorKey: ErrorRecoveryAttempter(recoveryAttempter: { (error, optionIndex) -> Bool in
					if optionIndex == 1 {
						NSWorkspace.shared().openFile(self.logFilePath, withApplication: "Console")
					}
					return true
				})
			]
			return .Failure(NSError(domain: "com.chain.ChainCore.createuser", code: 0, userInfo: userInfo))
		}
	}

    private func createDatabaseSync(name dbname: String) -> OperationStatus {
        let task = Process()
        task.launchPath = binPath.appending("/createdb")
        task.environment = [
            "PGHOST": "localhost"
        ]
        task.arguments = [
            "-p", String(port),
            dbname
        ]
        task.standardOutput = Pipe()
        let errorPipe = Pipe()
        task.standardError = errorPipe
        task.launch()
        let errorDescription = String(data: errorPipe.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? "(incorrectly encoded error message)"
        task.waitUntilExit()

        if task.terminationStatus == 0 {
            return .Success
        } else {
            let userInfo: [String: Any] = [
                NSLocalizedDescriptionKey: NSLocalizedString("Could not create database “\(dbname)”.", comment: ""),
                NSLocalizedRecoverySuggestionErrorKey: errorDescription,
                NSLocalizedRecoveryOptionsErrorKey: ["OK", "Open Server Log"],
                NSRecoveryAttempterErrorKey: ErrorRecoveryAttempter(recoveryAttempter: { (error, optionIndex) -> Bool in
                    if optionIndex == 1 {
                        NSWorkspace.shared().openFile(self.logFilePath, withApplication: "Console")
                    }
                    return true
                })
            ]
            return .Failure(NSError(domain: "com.chain.ChainCore.createdb", code: 0, userInfo: userInfo))
        }
    }
	
}


class Database: NSObject {
	dynamic var name: String = ""
	
	init(_ name: String) {
		super.init()
		self.name = name
	}
}

