package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"

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
		BaseURL string
		Latency map[string]interface{}
	}

	v.BaseURL = req.URL.Query().Get("baseurl")
	if v.BaseURL == "" {
		v.BaseURL = "http://localhost:1999/"
	}

	err := getDebugVars(v.BaseURL, &v)
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

func getDebugVars(baseURL string, v interface{}) error {
	resp, err := http.Get(strings.TrimRight(baseURL, "/") + "/debug/vars")
	if err != nil {
		return err
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

const indexHTML = `
<h1>perfdash</h1>
{{$b := .BaseURL}}
{{range $k, $v := .Latency}}
	<img src="/heatmap.png?name={{$k}}&baseurl={{$b}}">
{{end}}
`
