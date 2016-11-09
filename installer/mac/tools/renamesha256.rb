#!/usr/bin/env ruby
# Renames `filename.tar.gz` into `filename.tar.gz-<sha256digest>`
path = ARGV[0]
if !File.exist?(path)
    $stderr.puts "File not found: #{path}"
    exit(1)
end
digest = `shasum -a 256 #{path}`[0,64]

name = File.basename(path) + "-#{digest}"
newpath = File.dirname(path) + "/" + name
if system(%{mv "#{path}" "#{newpath}"})
    exit(0)
else
    exit(1)
end
