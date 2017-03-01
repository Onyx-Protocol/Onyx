import Cocoa

class Config: NSObject {
    static let shared: Config = Config()
    private(set) var corePort:UInt = 1999
    private(set) var dbPort:UInt   = 1998

    /// Checks if the port is in use by another process.
    static func portInUse(_ port: UInt) -> Bool {
        var output : [String] = []

        let task = Process()
        task.launchPath = "/usr/sbin/lsof"
        task.arguments = ["-n", "-i:\(port)"]

        let outpipe = Pipe()
        task.standardOutput = outpipe
        task.launch()

        let outdata = outpipe.fileHandleForReading.readDataToEndOfFile()
        if var string = String(data: outdata, encoding: .utf8) {
            string = string.trimmingCharacters(in: .newlines)
            output = string.components(separatedBy: "\n")
        }

        task.waitUntilExit()

        if (output.count > 0 && output[0] != "") {
            return true
        }
        return false
    }
}
