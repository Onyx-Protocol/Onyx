#!/usr/bin/env ruby
# This script takes the Chain Core.app file in its folder, zips it, versions it and updates the update file.

require 'json'
require 'fileutils'

TIME_REGEXP = /\d\d\d\d-\d\d-\d\d \d\d-\d\d-\d\d/
dirpath     = File.expand_path("../updates", File.dirname(__FILE__))
filename    = "Chain Core.app"
app_paths = Dir["#{dirpath}/**/Chain Core.app"].to_a.sort_by{|path| path[TIME_REGEXP] || "9999-99-99 99-99-99" }
app_path = app_paths.last # use the latest build
(app_paths - [app_path]).each do |path|
  if x = path[TIME_REGEXP]
    puts "path has time: #{path}: #{x} // #{File.dirname(path)}"
    FileUtils.rm_r(File.dirname(path))
  else
    FileUtils.rm_r(path)
  end
end
info        = JSON.load(`plutil -convert json -o - "#{app_path}/Contents/Info.plist"`)
updates_url = info["SUFeedURL"]
base_url = File.dirname(updates_url)
version     = info["CFBundleVersion"]
zipname     = "Chain_Core_#{version}.zip"
latestzipname  = "Chain_Core.zip"

updates_filename = updates_url[/updates.*\.xml$/]

system(%{rm -rf #{dirpath}/#{zipname}})
system(%{ditto -ck --keepParent --rsrc "#{app_path}" "#{dirpath}/#{zipname}"})

system("rm #{dirpath}/#{latestzipname}")
system("cp #{dirpath}/#{zipname} #{dirpath}/#{latestzipname}")

bytesize = `stat -f %z "#{dirpath}/#{zipname}"`.strip
pubdate  = `LC_TIME=en_US date +"%a, %d %b %G %T %z"`.strip

appcast_path = dirpath + "/" + updates_filename
appcast = File.read(appcast_path)
appcast.gsub!(%r{<link>.*?</link>}, %{<link>#{updates_url}</link>})
appcast.gsub!(%r{<title>.*?\d+\.\d+</title>}, %{<title>Chain Core Developer Edition #{version}</title>})
appcast.gsub!(%r{<pubDate>.*?</pubDate>}, %{<pubDate>#{pubdate}</pubDate>})
appcast.gsub!(%r{sparkle:version=".*?"}, %{sparkle:version="#{version}"})
appcast.gsub!(%r{url="[^"]*Chain[^"]*Core[^"]*\.zip"}, %{url="#{base_url}/#{zipname}"})
appcast.gsub!(%r{length="\d+"}, %{length="#{bytesize}"})
File.open(appcast_path, "w"){|f| f.write(appcast) }
