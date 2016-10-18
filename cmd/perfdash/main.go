package main

import (
	"html/template"
	"log"
	"net/http"

	"chain/env"
)

var addr = env.String("LISTEN", ":8080")

func main() {
	env.Parse()
	http.HandleFunc("/", index)
	http.HandleFunc("/heatmap.png", heatmap)
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

	baseURL := req.URL.Query().Get("baseurl")
	if baseURL == "" {
		baseURL = "http://localhost:1999/"
	}

	var err error
	v.ID, v.DebugVars, err = fetchDebugVars(baseURL)
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
{{$id := .ID}}
{{range $k, $v := .DebugVars.Latency}}
	<img src="/heatmap.png?name={{$k}}&id={{$id}}">
{{end}}
`
