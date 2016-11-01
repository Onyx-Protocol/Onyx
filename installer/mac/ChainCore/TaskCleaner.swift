import Foundation

// Note: this is not the most efficient way to watch the parent's death.
// This will be more efficient, but requires more complicated Xcode setup with extra binaries: 
// https://developer.apple.com/reference/corefoundation/1667011-cffiledescriptor
class TaskCleaner {

    public let childPid:  Int32
    public let parentPid: Int32
    public let interval:  Int

    private var task:Process? = nil

    init(childPid: Int32, parentPid: Int32 = ProcessInfo.processInfo.processIdentifier, interval: Int = 1) {
        self.parentPid = parentPid
        self.childPid = childPid
        self.interval = interval
    }

    func watch() {
        if task != nil {
            task?.terminate()
            task = nil
        }
        let script = "while kill -0 \(parentPid); do sleep \(interval); done; kill -9 \(childPid); exit 1"

        task = Process()
        task?.launchPath = "/bin/sh"
        task?.arguments = ["-c", script]
        task?.launch()
    }

    func terminate() {
        task?.terminate()
        task = nil
    }

}
