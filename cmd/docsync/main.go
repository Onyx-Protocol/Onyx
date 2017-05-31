package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func main() {
	localKeys := mustListContents("/Users/jeff/go/src/chain/cmd/docsync/DO_NOT_COMMIT")
	fmt.Println(len(localKeys))

	region := "us-east-1"
	sess := session.Must(session.NewSession(aws.NewConfig().WithRegion(region)))
	svc := s3.New(sess)

	var remoteKeys []string
	err := svc.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String("chain.com"),
		Prefix: aws.String("docs/"),
	}, func(page *s3.ListObjectsOutput, lastPage bool) bool {
		for _, obj := range page.Contents {
			remoteKeys = append(remoteKeys, *obj.Key)
		}
		return true
	})
	if err != nil {
		log.Fatalln("s3 list objects error:", err)
	}

	intersection, localOnly, remoteOnly := setAnalyze(localKeys, remoteKeys)
	fmt.Println("local size:", len(localKeys))
	fmt.Println("remote size:", len(remoteKeys))
	fmt.Println("intersection:", len(intersection))
	fmt.Println("local only:", len(localOnly))
	fmt.Println("remote only:", len(remoteOnly))
}

func setAnalyze(a, b []string) (intersection []string, aOnly []string, bOnly []string) {
	sort.Strings(append([]string{}, a...))
	sort.Strings(append([]string{}, b...))

	fmt.Println(a[0])
	fmt.Println(b[0])

	var i, j int

	for {
		if i == len(a) {
			bOnly = append(bOnly, b[j:]...)
			break
		}

		if j == len(b) {
			aOnly = append(aOnly, a[i:]...)
			break
		}

		if a[i] < b[j] {
			aOnly = append(aOnly, a[i])
			i++
			continue
		}

		if b[j] < a[i] {
			bOnly = append(bOnly, b[j])
			j++
			continue
		}

		// invariant: a[i] == b[j]
		intersection = append(intersection, a[i])
		i++
		j++
	}

	return
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
			res = append(res, n)
		}
	}

	return res
}
