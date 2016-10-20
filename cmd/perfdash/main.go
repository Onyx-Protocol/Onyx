package main

import (
	"html/template"
	"log"
	"net/http"

	"chain/env"
)

var (
	addr    = env.String("LISTEN", ":8080")
	baseURL = env.String("BASE_URL", "http://localhost:1999/")
	token   = env.String("CLIENT_ACCESS_TOKEN", "")
)

func main() {
	env.Parse()
	http.HandleFunc("/", index)
	http.HandleFunc("/histogram.png", histogram)
	log.Fatalln(http.ListenAndServe(*addr, nil))
}

var tmpl = template.Must(template.New("index.html").Parse(indexHTML))

func index(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}

	var v struct {
		DebugVars *debugVars
		ID        int
	}

	u := req.URL.Query().Get("baseurl")
	if u == "" {
		u = *baseURL
	}

	var err error
	v.ID, v.DebugVars, err = fetchDebugVars(u, *token)
	if err != nil {
		http.Error(w, err.Error(), 500)
		log.Println(err)
		return
	}

	err = tmpl.Execute(w, v)
	if err != nil {
		log.Println(err)
	}
}

const indexHTML = `
<h1>perfdash</h1>
Open dev tools to see the full /debug/vars data.
{{$id := .ID}}
{{range $k, $v := .DebugVars.Latency}}
	<img src="/histogram.png?name={{$k}}&id={{$id}}">
{{end}}
<script>
console.log("/debug/vars", {{.DebugVars.Raw}});
</script>
`
