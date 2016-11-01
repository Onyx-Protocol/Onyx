import Foundation

extension FileManager {
	func applicationSupportDirectoryPath() -> String {
		let url = self.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
		
		let bundleName = Bundle.main.object(forInfoDictionaryKey: kCFBundleNameKey as String) as! String
		let path = url.appendingPathComponent(bundleName).path
		
		if !self.fileExists(atPath: path) {
			try! self.createDirectory(atPath: path, withIntermediateDirectories: false)
		}
		
		return path
	}
}
