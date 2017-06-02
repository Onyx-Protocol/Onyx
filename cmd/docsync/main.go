package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func main() {
	localKeys := mustListContents(os.Args[2])

	region := "us-east-1"
	sess := session.Must(session.NewSession(aws.NewConfig().WithRegion(region)))
	svc := s3.New(sess)

	var remoteKeys []string
	err := svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(os.Args[1]),
		Prefix: aws.String("docs/"),
	}, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, obj := range page.Contents {
			if !(*obj.Key == "docs/") {
				remoteKeys = append(remoteKeys, *obj.Key)
			}
		}
		return true
	})

	if err != nil {
		log.Fatalln("s3 list objects error:", err)
	}

	var prefixedLocalKeys []string
	for _, k := range localKeys {
		prefixedLocalKeys = append(prefixedLocalKeys, path.Join("docs", k))
	}
	fmt.Println("keys to upload:", len(prefixedLocalKeys)) // TEMP

	remoteOnly := setDiff(remoteKeys, prefixedLocalKeys)
	fmt.Println("keys to delete:", len(remoteOnly)) // TEMP

	for _, k := range prefixedLocalKeys {
		var body []byte

		path := strings.Replace(k, "docs", os.Args[2], 1)
		body, err = ioutil.ReadFile(path)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		upload := &s3.PutObjectInput{
			Bucket: aws.String(os.Args[1]),
			Key:    aws.String(k),
			Body:   bytes.NewReader(body),
		}

		ext := filepath.Ext(path)
		contentType := mime.TypeByExtension(ext)

		if contentType == "" {
			upload.SetContentType("text/html")
		} else {
			upload.SetContentType(contentType)
		}

		fmt.Println("uploading ", k, " with type ", contentType)

		_, err = svc.PutObject(upload)

		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	for _, k := range remoteOnly {
		remove := &s3.DeleteObjectInput{
			Bucket: aws.String(os.Args[1]),
			Key:    aws.String(k),
		}

		fmt.Println("deleting ", k)

		_, err = svc.DeleteObject(remove)

		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}
}

func setDiff(a, b []string) []string {
	// don't modify input
	a = append([]string{}, a...)
	b = append([]string{}, b...)
	sort.Strings(a)
	sort.Strings(b)

	var (
		diff []string
		i, j int
	)

	for {
		if i == len(a) {
			break
		}

		if j == len(b) {
			diff = append(diff, a[i:]...)
			break
		}

		if a[i] < b[j] {
			diff = append(diff, a[i])
			i++
			continue
		}

		if b[j] < a[i] {
			j++
			continue
		}

		// invariant: a[i] == b[j]
		i++
		j++
	}

	return diff
}

func mustListContents(parentPath string) []string {
	files, err := ioutil.ReadDir(parentPath)
	if err != nil {
		log.Fatalln("ReadDir error:", err)
	}

	var res []string
	for _, f := range files {
		n := f.Name()

		if f.IsDir() {
			descendants := mustListContents(path.Join(parentPath, n))
			for _, d := range descendants {
				res = append(res, path.Join(n, d))
			}
		} else {
			if !strings.HasPrefix(n, ".") {
				res = append(res, n)
			}
		}
	}

	return res
}
