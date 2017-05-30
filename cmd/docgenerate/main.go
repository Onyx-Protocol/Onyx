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

	// Generate guides and reference docs
	mustRunIn(path.Join(srcdir, "docs"), "md2html", "build", outdir)

	// Generate SDK-specific documentation
	wg := new(sync.WaitGroup)
	for _, v := range versionPaths(srcdir) {
		wg.Add(1)
		go makeSdkDocs(wg, v, srcdir, outdir)
	}
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
