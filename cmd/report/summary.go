package main

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
)

type results struct {
	Runs []Run
}

type Run struct {
	StartedAt string
	Elapsed   time.Duration
	Path      string
	Command   []string
	Ok        bool
}

func getSummary(ctx context.Context) *results {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	res, err := s3svc.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: bucket,
		Key:    aws.String("results.json"),
	})
	if a, ok := err.(awserr.Error); ok && a.Code() == "NoSuchKey" {
		return new(results)
	} else if err != nil {
		panic(err)
	}
	var s *results
	err = json.NewDecoder(res.Body).Decode(&s)
	if err != nil {
		panic(err)
	}
	return s
}

func putSummary(ctx context.Context, s *results) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	b, err := json.MarshalIndent(s, "", "\t")
	if err != nil {
		panic(err)
	}
	put(ctx, "results.json", b, "text/plain; charset=utf-8")

	reverse(s.Runs)

	var buf bytes.Buffer
	err = summaryTemplate.Execute(&buf, s)
	if err != nil {
		panic(err)
	}
	put(ctx, "index.html", buf.Bytes(), "text/html; charset=utf-8")

	_, err = s3svc.PutBucketWebsite(&s3.PutBucketWebsiteInput{
		Bucket: bucket,
		WebsiteConfiguration: &s3.WebsiteConfiguration{
			IndexDocument: &s3.IndexDocument{Suffix: aws.String("index.html")},
		},
	})
	if err != nil {
		panic(err)
	}
}

func reverse(a []Run) {
	for i := 0; i < len(a)/2; i++ {
		a[i], a[len(a)-i-1] = a[len(a)-i-1], a[i]
	}
}

var summaryTemplate = template.Must(template.New("results").Parse(`
<!doctype html>
<html>
<head>
<style>
:root {
	font-family: monospace;
}
td {
	padding: .25em .5em;
}
</style>
</head>
<body>

<p>Raw results: <a href=results.json>results.json</a></p>

<table>
{{range .Runs}}
<tr>
<td>{{.StartedAt}}</td>
<td>{{if .Ok}}ok{{else}}<a href={{.Path}}><b>fail</b></a>{{end}}</td>
<td>{{.Elapsed}}</td>
<td>{{.Command}}</td>
</tr>
{{ end }}
</table>
</body>
</html>
`))
