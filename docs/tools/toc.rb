Levels = [2,3]

basepath = File.expand_path("..", File.dirname(__FILE__))
Dir["#{basepath}/**/*.md"].each do |mdfile|
  puts "TOC for #{mdfile}:\n\n"
  File.read(mdfile).each_line do |line|
    if line =~ /^(#+)\s(.*)/
      prefix = $1
      title = $2
      depth = prefix.size
      if depth == 1
        puts "# #{title}\n\n"
      end
      if Levels.include?(depth)
        s = ""
        s << "  "*(depth-2)
        anchor = title.downcase.gsub(/\W+/,"-").gsub(/(\d)\-(\d)/,"\\1\\2")
        s << "* [#{title}](##{anchor})"
        puts s
      end
    end
  end
  puts "\n\n---\n\n\n"
end