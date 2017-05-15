package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	bucket = aws.String(os.Getenv("BUCKET"))
	region = "us-east-1" // TODO(kr): figure out how to not hard code this
	sess   = session.Must(session.NewSession(aws.NewConfig().WithRegion(region)))
	s3svc  = s3.New(sess)
)

func main() {
	t := time.Now().UTC()
	s := Run{
		StartedAt: t.Format(time.RFC3339),
		Command:   os.Args[1:],
	}
	s.Path = "log/" + s.StartedAt + "-" + slug(strings.Join(s.Command, "-"))
	var b bytes.Buffer
	fmt.Fprintln(&b, s.StartedAt)
	fmt.Fprintf(&b, "%q\n", s.Command)
	s.Ok = run(&b, s.Command)
	s.Elapsed = time.Now().Sub(t)
	fmt.Fprintln(&b, "elapsed", s.Elapsed)
	// Do all the S3 stuff at the end.
	// If anything fails after this point,
	// we will panic.
	ctx := context.Background()
	report(ctx, b.Bytes(), s)
}

func run(w io.Writer, args []string) bool {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = w
	cmd.Stderr = w
	err := cmd.Run()
	w.Write([]byte{'\n'}) // output might not have trailing LF
	if err != nil {
		fmt.Fprintln(w, err)
	}
	return err == nil
}

func report(ctx context.Context, body []byte, s Run) {
	put(ctx, s.Path, body, "text/plain; charset=utf-8")
	summarize(ctx, s)
	put(ctx, "lastrun", []byte(s.StartedAt), "text/plain; charset=utf-8")
}

func summarize(ctx context.Context, s Run) {
	sum := getSummary(ctx)
	sum.Runs = append(sum.Runs, s)
	putSummary(ctx, sum)
}

func put(ctx context.Context, key string, body []byte, contentType string) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	_, err := s3svc.PutObjectWithContext(ctx, &s3.PutObjectInput{
		ACL:         aws.String("public-read"),
		Bucket:      bucket,
		Key:         &key,
		Body:        bytes.NewReader(body),
		ContentType: &contentType,
	})
	if err != nil {
		panic(err)
	}
}

func slug(s string) string {
	return strings.Map(slugChar, s)
}

func slugChar(c rune) rune {
	if !okInURL(c) {
		return -1
	}
	return c
}

func okInURL(c rune) bool {
	// See RFC 3986 (URI Syntax) section 2.3 (Unreserved Characters)
	// https://www.ietf.org/rfc/rfc3986.txt
	return strings.ContainsRune("-._~", c) ||
		'a' <= c && c <= 'z' ||
		'A' <= c && c <= 'Z' ||
		'0' <= c && c <= '9'
}
