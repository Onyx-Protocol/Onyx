#!/usr/bin/env ruby
# Based on https://blogs.oracle.com/dipol/entry/dynamic_libraries_rpath_and_mac

def main
  dirpath = File.expand_path("..", File.dirname(__FILE__))
  Dir["#{dirpath}/pg/build/bin/**/*"].each do |path|
    if path !~ /-changed$/
      patch_binary(path, dirpath)
    end
  end
  # Dir["#{dirpath}/pg/build/lib/**/*.so"].each do |path|
  #   if path !~ /-changed$/
  #     patch_binary(path, dirpath)
  #   end
  # end
  Dir["#{dirpath}/pg/build/lib/*.dylib"].each do |path|
    if path !~ /-changed$/
      patch_binary(path, dirpath)
    end
  end
  
  puts "\n\n-----------------------------\n\n"
  
  Dir["#{dirpath}/pg/build/bin/**/*"].each do |path|
    if path !~ /-changed$/
    else
      inspect_binary(path, dirpath)
    end
  end
  Dir["#{dirpath}/pg/build/lib/*.dylib-changed"].each do |path|
    if path !~ /-changed$/
    else
      inspect_binary(path, dirpath)
    end
  end
end

def patch_binary(path, dirpath)
  return if File.directory?(path)
  return if File.symlink?(path)
  x = File.open(path,"r"){|f|f.read(2)}
  if x == "#!"
    # this is a shell script, ignore.
    return
  end
  
  puts ">> #{path}"

  `rm -f #{path}-changed`
  # `cp #{path} #{path}-changed`
  # path = path + "-changed"
  
  # 1. Change the install name itself (applies only to libs, not executables)
  ids = `otool -D #{path}`.strip.split("\n")[1..-1].join("\n") # removes prefix "this-filename:\n"
  ids.scan(%r{#{dirpath}/pg/src/\.\./build/lib/.*\.(?:dylib|so)}) do |id|
    libname = id.gsub(%r{#{dirpath}/pg/src/\.\./build/lib/},"")
    cmd = %{install_name_tool -id "@loader_path/../lib/#{libname}" #{path}}
    puts "    $ #{cmd}"
    system("chmod +w #{path}")
    system(cmd)
    system("chmod -w #{path}")
  end

  # 2. Get dependencies
  deps = `otool -L #{path}`.strip.split("\n")[1..-1].join("\n") # removes prefix "this-filename:\n"
  deps.scan(%r{#{dirpath}/pg/src/\.\./build/lib/.*\.(?:dylib|so)}) do |deppath|
    libname = deppath.gsub(%r{#{dirpath}/pg/src/\.\./build/lib/},"")
    relpath = if path[%r{/build/lib/}]
      "@loader_path"
    else
      "@loader_path/../lib"
    end
    cmd = %{install_name_tool -change "#{deppath}" "#{relpath}/#{libname}" #{path}}
    puts "    $ #{cmd}"
    puts "    dep: #{relpath}/#{libname}"
    system("chmod +w #{path}")
    system(cmd)
    system("chmod -w #{path}")
  end
end


def inspect_binary(path, dirpath)
  return if File.directory?(path)
  return if File.symlink?(path)
  x = File.open(path,"r"){|f|f.read(2)}
  if x == "#!"
    # this is a shell script, ignore.
    return
  end
  
  puts "INSPECT #{path}"
    
  # 1. Change the install name itself (applies only to libs, not executables)
  ids = `otool -D #{path}`.strip.split("\n")[1..-1].join("\n") # removes prefix "this-filename:\n"
  ids.split("\n").each do |id|
    puts "    id: #{id.strip}"
  end

  # 2. Get dependencies
  deps = `otool -L #{path}`.strip.split("\n")[1..-1].join("\n") # removes prefix "this-filename:\n"
  deps.split("\n").each do |deppath|
    puts "    dep: #{deppath}"
  end
end

main
