#!/usr/bin/env ruby
# This script takes the Chain Core.app file in its folder, zips it, versions it and updates the update file.

require 'json'

base_url = "https://download.chain.com/mac"
dirpath = File.expand_path("../updates", File.dirname(__FILE__))
filename = "Chain Core.app"
version = JSON.load(`plutil -convert json -o - "#{dirpath}/#{filename}/Contents/Info.plist"`)["CFBundleVersion"]
zipname = "Chain_Core_#{version}.zip"
latestname = "Chain_Core.zip"

system(%{rm -rf #{dirpath}/#{zipname}})
system(%{ditto -ck --keepParent --rsrc "#{dirpath}/#{filename}" "#{dirpath}/#{zipname}"})

system("rm #{dirpath}/#{latestname}")
system("cp #{dirpath}/#{zipname} #{dirpath}/#{latestname}")

bytesize = `stat -f %z "#{dirpath}/#{zipname}"`.strip
pubdate  = `LC_TIME=en_US date +"%a, %d %b %G %T %z"`.strip

appcast_path = dirpath + "/updates.xml"
appcast = File.read(appcast_path)
appcast.gsub!(%r{<link>.*?</link>}, %{<link>#{base_url}/updates.xml</link>})
appcast.gsub!(%r{<title>.*?\d+\.\d+</title>}, %{<title>Chain Core Developer Edition #{version}</title>})
appcast.gsub!(%r{<pubDate>.*?</pubDate>}, %{<pubDate>#{pubdate}</pubDate>})
appcast.gsub!(%r{sparkle:version=".*?"}, %{sparkle:version="#{version}"})
appcast.gsub!(%r{url="[^"]*Chain[^"]*Core[^"]*\.zip"}, %{url="#{base_url}/#{zipname}"})
appcast.gsub!(%r{length="\d+"}, %{length="#{bytesize}"})
File.open(appcast_path, "w"){|f| f.write(appcast) }
