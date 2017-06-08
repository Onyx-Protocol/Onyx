// Command docgenerate generates static documentation for Chain Core and the
// Chain Core SDKs.
//
// Usage:
//
//     docgenerate chainPath outputDir
//
// where chainPath is a path to your Chain repository, and outputDir is a
// target directory for the static files.
//
// Before running docgenerate, ensure the following:
// - The md2html command (also in this repo) is installed and up-to-date.
// - Your historical version branches (e.g., 1.0-stable) are up-to-date.
//   docgenerate uses these branches to generate SDK documentation, and will not
//   fetch from a git remote.
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Protects calls to bundle-install
var bundleInstallMutex = new(sync.Mutex)

func main() {
	var err error

	srcdir := os.Args[1]
	srcdir, err = filepath.Abs(srcdir)
	if err != nil {
		panic(err)
	}

	outdir := os.Args[2]
	outdir, err = filepath.Abs(outdir)
	if err != nil {
		panic(err)
	}

	wg := new(sync.WaitGroup)

	// Generate SDK-specific documentation and search index files
	for _, v := range versionPaths(srcdir) {
		wg.Add(1)
		go makeSdkDocs(wg, v, srcdir, outdir)
		wg.Add(1)
		go makeIndexInputFiles(wg, v, srcdir) // Note: this updates your local searchIndex.js file in $CHAIN in order to be copied out in the following md2html call.
	}

	// Generate guides and reference docs
	mustRunIn(path.Join(srcdir, "docs"), "md2html", "build", outdir)

	wg.Wait()
}

func makeSdkDocs(wg *sync.WaitGroup, version, srcdir, docPath string) {
	defer wg.Done()

	d := makeTempRepo(srcdir)
	defer func() {
		os.RemoveAll(d)
	}()

	repoPath := path.Join(d, "src", "chain")
	docVersionPath := path.Join(docPath, version)

	if err := runIn(repoPath, "git", "checkout", "-q", version+"-stable"); err != nil {
		fmt.Printf("error making SDK docs for %s: %v\n", version, err)
		return
	}

	wg.Add(1)
	makeJavaDoc(wg, repoPath, docVersionPath)

	wg.Add(1)
	makeNodeDoc(wg, repoPath, docVersionPath)

	wg.Add(1)
	makeRubyDoc(wg, repoPath, docVersionPath)
}

func makeJavaDoc(wg *sync.WaitGroup, repoPath, docVersionPath string) {
	defer wg.Done()

	sdkpath := path.Join(repoPath, "sdk", "java")
	outdir := path.Join(docVersionPath, "java", "javadoc")

	mustRunIn(sdkpath, "mvn", "javadoc:javadoc")
	mustRun("mkdir", "-p", outdir)
	mustRunIn(sdkpath, "cp", "-R", "target/site/apidocs/", outdir)
}

func makeRubyDoc(wg *sync.WaitGroup, repoPath, docVersionPath string) {
	defer wg.Done()

	sdkPath := path.Join(repoPath, "sdk", "ruby")
	outdir := path.Join(docVersionPath, "ruby", "doc")

	bundleInstallMutex.Lock()
	mustRunIn(sdkPath, "bundle", "install")
	bundleInstallMutex.Unlock()

	mustRunIn(sdkPath, "bundle", "exec", "yardoc")
	mustRun("mkdir", "-p", outdir)
	mustRunIn(sdkPath, "cp", "-R", "doc/", outdir)
}

func makeNodeDoc(wg *sync.WaitGroup, repoPath, docVersionPath string) {
	defer wg.Done()

	sdkPath := path.Join(repoPath, "sdk", "node")
	outdir := path.Join(docVersionPath, "node", "doc")

	mustRunIn(sdkPath, "npm", "set", "progress=false") // Note: this will clobber host settings. This should be NBD, especially when we run from Docker.
	mustRunIn(sdkPath, "npm", "install", "--quiet")
	mustRunIn(sdkPath, "npm", "run", "docs")
	mustRun("mkdir", "-p", outdir)
	mustRunIn(sdkPath, "cp", "-R", "doc/", outdir)
}

func makeTempRepo(srcdir string) string {
	d, err := ioutil.TempDir("/tmp", "docgenerate")
	if err != nil {
		log.Fatalln("could not create temp directory:", err)
	}

	chainDir := path.Join(d, "src", "chain")
	mustRun("mkdir", "-p", chainDir)
	mustRun("git", "clone", "-q", srcdir, chainDir)

	return d
}

func makeIndexInputFiles(wg *sync.WaitGroup, version, srcdir string) {
	var err error

	defer wg.Done()

	jsPath := path.Join(srcdir, "docs", version, "searchIndex.js")

	_, err = os.Create(jsPath)
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile(jsPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = f.WriteString("window.searchIndex = [")
	if err != nil {
		panic(err)
	}

	versionPath := path.Join(srcdir, "docs", version)
	srcPath := path.Join(srcdir, "docs")
	contents := createIndexFile(versionPath, srcPath, version)

	_, err = f.WriteString(contents)
	if err != nil {
		panic(err)
	}

	_, err = f.WriteString("]")
	if err != nil {
		panic(err)
	}
}

func mustRun(command string, args ...string) {
	c := exec.Command(command, args...)
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		log.Fatalln("command failed:", command, strings.Join(args, " "), "\n", err)
	}
}

func runIn(dir, command string, args ...string) error {
	c := exec.Command(command, args...)
	c.Dir = dir
	c.Stderr = os.Stderr
	return c.Run()
}

func mustRunIn(dir, command string, args ...string) {
	if err := runIn(dir, command, args...); err != nil {
		log.Fatalln("command failed:", command, strings.Join(args, " "), "\n", err)
	}
}

func versionPaths(srcdir string) []string {
	fis, err := ioutil.ReadDir(path.Join(srcdir, "docs"))
	if err != nil {
		panic(err)
	}

	var paths []string
	for _, fi := range fis {
		if !fi.IsDir() || !regexp.MustCompile("^\\d+\\.\\d+$").MatchString(fi.Name()) {
			continue
		}
		paths = append(paths, fi.Name())
	}

	return paths
}

func createIndexFile(parentPath string, srcPath string, version string) string {

	files, err := ioutil.ReadDir(parentPath)
	if err != nil {
		log.Fatalln("ReadDir error:", err)
	}

	var contents []string

	for _, f := range files {
		n := f.Name()

		if strings.HasPrefix(n, ".") {
			continue
		}

		if f.IsDir() {
			if n == "enterprise" {
				continue
			}
			contents = append(contents, createIndexFile(path.Join(parentPath, n), srcPath, version))
		} else {
			ext := filepath.Ext(n)
			if ext == ".md" {
				if n == "license.md" {
					continue
				}

				tempPath := path.Join(parentPath, n)
				tempFile, err := ioutil.ReadFile(tempPath)
				if err != nil {
					panic(err)
				}

				contents = append(contents, generateJSONFromBytes(tempFile, tempPath))
			}
		}
	}
	return strings.Join(contents, ",")
}

func generateJSONFromBytes(tempFile []byte, tempPath string) string {
	type Index struct {
		URL     string
		Body    string
		Title   string
		Snippet string
	}

	tempString := string(tempFile)
	lines := strings.Split(tempString, "\n")
	title := ""
	snippet := ""

	if lines[0] == "<!---" {
		snippet = lines[1]
	}

	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "# ") {
			title = strings.TrimPrefix(lines[i], "# ")
			break
		}
	}

	urlSlice := strings.Split(tempPath, "src/chain")
	url := strings.TrimSuffix(urlSlice[len(urlSlice)-1], ".md")
	indexed := &Index{URL: url, Body: tempString, Title: title, Snippet: snippet}
	b, err := json.Marshal(indexed)
	if err != nil {
		panic(err)
	}

	return string(b)
}
