import Cocoa

class Config: NSObject {
    static let shared: Config = Config()
    private(set) var corePort:UInt = 1999
    private(set) var dbPort:UInt   = 1998

    /// Checks if the port is in use by another process.
    static func portInUse(_ port: UInt) -> Bool {
        let sock = socket(PF_INET, SOCK_STREAM, IPPROTO_TCP)
        if sock <= 0 {
            return false
        }

        var listenAddress = sockaddr_in()
        listenAddress.sin_family = UInt8(AF_INET)
        listenAddress.sin_port = in_port_t(port).bigEndian
        listenAddress.sin_len = UInt8(MemoryLayout<sockaddr_in>.size)
        listenAddress.sin_addr.s_addr = inet_addr("127.0.0.1")

        let bindRes = withUnsafePointer(to: &listenAddress) { (sockaddrPointer: UnsafePointer<sockaddr_in>) in
            sockaddrPointer.withMemoryRebound(to: sockaddr.self, capacity: 1) { (sockaddrPointer2: UnsafePointer<sockaddr>) in
                Darwin.bind(sock, sockaddrPointer2, socklen_t(MemoryLayout<sockaddr_in>.stride))
            }
        }

        let bindErr = Darwin.errno
        close(sock)

        if bindRes == -1 && bindErr == EADDRINUSE {
            return true
        }
        
        return false
    }
}
