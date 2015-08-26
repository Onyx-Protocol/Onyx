package secureheader_test

import (
	"github.com/kr/secureheader"
	"net/http"
)

func Example() {
	http.Handle("/", http.FileServer(http.Dir("/tmp")))
	http.ListenAndServe(":80", secureheader.DefaultConfig)
}

func Example_custom() {
	http.Handle("/", http.FileServer(http.Dir("/tmp")))
	secureheader.DefaultConfig.HSTSIncludeSubdomains = false
	secureheader.DefaultConfig.FrameOptions = false
	http.ListenAndServe(":80", secureheader.DefaultConfig)
}
