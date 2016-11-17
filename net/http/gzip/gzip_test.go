package gzip

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

var (
	small  = []byte(`{"message":"ok"}`)
	medium = []byte(`{"id":"961458e16018cb60f06b01d303ae0d8e2b3ff98698a1f80b5c6715969644f519","timestamp":"2016-10-04T19:13:23Z","block_id":"181c11b24c7dbdd5ce5e2b9da1b665878a80712dbfd1796613e66305da49ca7c","block_height":2,"position":0,"reference_data":{},"is_local":"yes","inputs":[{"action":"issue","asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":5,"issuance_program":"027b7d75766baa205fd08ca9e18b180c7da3ace70e890cba8c7014c7da5c9ed78e3d9e253cccec8f5151ad696c00c0","reference_data":{},"is_local":"yes"}],"outputs":[{"action":"control","purpose":"receive","position":0,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":5,"account_id":"acc0KNY9W8QG0802","account_alias":"t","account_tags":null,"control_program":"766baa20107c8767129a4ae5325371946e44f4ae76448452f722768048bd2d5cf12fc1595151ad696c00c0","reference_data":{},"is_local":"yes"}]}`)
	large  = []byte(`{"items":[{"id":"0266cf2ed4ff3cb989341a9f9b2c3e7ffcdf2133ee84df9652834ca97a9bfe53","timestamp":"2016-10-04T20:31:02Z","block_id":"c77d0004600ce1de5ac1c815dba6fd3512b396292203e96b56c3e618a8c07113","block_height":8,"position":0,"reference_data":{},"is_local":"yes","inputs":[{"action":"spend","asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":20,"spent_output":{"position":1,"transaction_id":"092de8cc56abcd588919973c2eb5b5a56355e4d4d50910f5bcfa25ca4e2c0124"},"account_id":"acc0KP0F1K9G081A","account_alias":"foo","account_tags":null,"reference_data":{},"is_local":"yes"}],"outputs":[{"action":"control","purpose":"change","position":0,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":17,"account_id":"acc0KP0F1K9G081A","account_alias":"foo","account_tags":null,"control_program":"766baa2097abd5c16fab864da84605e5cc2ae96e946949c63470658f3b34a10e8594bc485151ad696c00c0","reference_data":{},"is_local":"yes"},{"action":"retire","position":1,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":3,"control_program":"6a","reference_data":{},"is_local":"no"}]},{"id":"712e79e954150750db4e245112429e18de7e67465858074ef79b5a83621d52f7","timestamp":"2016-10-04T20:30:37Z","block_id":"9a2e4a3f8a837935ef7c1d40e0032922fcdbf1e5e152222e7f2b9b5695170b03","block_height":7,"position":0,"reference_data":{},"is_local":"yes","inputs":[{"action":"issue","asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":20,"issuance_program":"027b7d75766baa205fd08ca9e18b180c7da3ace70e890cba8c7014c7da5c9ed78e3d9e253cccec8f5151ad696c00c0","reference_data":{},"is_local":"yes"}],"outputs":[{"action":"retire","position":0,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":20,"control_program":"6a","reference_data":{},"is_local":"no"}]},{"id":"092de8cc56abcd588919973c2eb5b5a56355e4d4d50910f5bcfa25ca4e2c0124","timestamp":"2016-10-04T20:30:13Z","block_id":"bf38fe464a50a24e90ecf01b75101c7063c94c7ac6e4ec9d349ced0033c6b233","block_height":6,"position":0,"reference_data":{},"is_local":"yes","inputs":[{"action":"spend","asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":55,"spent_output":{"position":0,"transaction_id":"8c851f25563a33b79a3f30139c88854c607979db437f71b31549c664f6995113"},"account_id":"acc0KNY9W8QG0802","account_alias":"t","account_tags":null,"reference_data":{},"is_local":"yes"}],"outputs":[{"action":"control","purpose":"change","position":0,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":35,"account_id":"acc0KNY9W8QG0802","account_alias":"t","account_tags":null,"control_program":"766baa20ae40f6e7509b6a86deab669f36b3a8bd43a4d6128904e97ebabecb81473084a65151ad696c00c0","reference_data":{},"is_local":"yes"},{"action":"control","purpose":"receive","position":1,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":20,"account_id":"acc0KP0F1K9G081A","account_alias":"foo","account_tags":null,"control_program":"766baa20a2abff2f62e5a9912bc9776c170f66219077adab278ca683b4583b7a9d6c83b85151ad696c00c0","reference_data":{},"is_local":"yes"}]},{"id":"4e6ad8970866d652f755c32fd04868ecd63d292021b3961ec4a87d9d8cd97ccc","timestamp":"2016-10-04T20:29:26Z","block_id":"033e802edc6d4b22c17679a0a1512dc04448b13dc2b2194c58ec11ac4c4c4cab","block_height":5,"position":0,"reference_data":{},"is_local":"yes","inputs":[{"action":"issue","asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":1,"issuance_program":"027b7d75766baa205fd08ca9e18b180c7da3ace70e890cba8c7014c7da5c9ed78e3d9e253cccec8f5151ad696c00c0","reference_data":{},"is_local":"yes"}],"outputs":[{"action":"control","purpose":"receive","position":0,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":1,"account_id":"acc0KP0F1K9G081A","account_alias":"foo","account_tags":null,"control_program":"766baa205dfbb62393e5e6d3b2a0772dd8fc1f865a0c774c44ee7de1bb40b18de5368bf55151ad696c00c0","reference_data":{},"is_local":"yes"}]},{"id":"8599e2feb681b84ae9e8233183c73b8a5bbf7564a9f7061b926f6b7040824608","timestamp":"2016-10-04T20:28:42Z","block_id":"cb6c384181883422e42f87637373593f7821df24723b89fb77f10fef69a73235","block_height":4,"position":0,"reference_data":{},"is_local":"yes","inputs":[{"action":"spend","asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":44,"spent_output":{"position":1,"transaction_id":"8c851f25563a33b79a3f30139c88854c607979db437f71b31549c664f6995113"},"account_id":"acc0KNY9W8QG0802","account_alias":"t","account_tags":null,"reference_data":{},"is_local":"yes"}],"outputs":[{"action":"control","purpose":"change","position":0,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":39,"account_id":"acc0KNY9W8QG0802","account_alias":"t","account_tags":null,"control_program":"766baa20d16227a885f913a958ff7e9df91118874133aba484175da7fb5e24b4bf6710315151ad696c00c0","reference_data":{},"is_local":"yes"},{"action":"control","purpose":"receive","position":1,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":5,"account_id":"acc0KP0F1K9G081A","account_alias":"foo","account_tags":null,"control_program":"766baa20153e9de41ba01abc10d5e7dbe5c3c222733c653cf8520a984802e7eaf18ba7855151ad696c00c0","reference_data":{},"is_local":"yes"}]},{"id":"8c851f25563a33b79a3f30139c88854c607979db437f71b31549c664f6995113","timestamp":"2016-10-04T20:23:27Z","block_id":"eb2593b3a13b386eab0dcc9f4c8aa5d03e01696d866e06c9048b7786127c163a","block_height":3,"position":0,"reference_data":{},"is_local":"yes","inputs":[{"action":"issue","asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":99,"issuance_program":"027b7d75766baa205fd08ca9e18b180c7da3ace70e890cba8c7014c7da5c9ed78e3d9e253cccec8f5151ad696c00c0","reference_data":{},"is_local":"yes"}],"outputs":[{"action":"control","purpose":"receive","position":0,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":55,"account_id":"acc0KNY9W8QG0802","account_alias":"t","account_tags":null,"control_program":"766baa209997e49055a4e9b020c3c2342a632b0977f8020778d3607acceacd5f0f8fc7fe5151ad696c00c0","reference_data":{},"is_local":"yes"},{"action":"control","purpose":"receive","position":1,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":44,"account_id":"acc0KNY9W8QG0802","account_alias":"t","account_tags":null,"control_program":"766baa20da29a78752723e4c873e1c46eafc0dfd22041fe2e818ed5584c9b3139fc5363a5151ad696c00c0","reference_data":{},"is_local":"yes"}]},{"id":"961458e16018cb60f06b01d303ae0d8e2b3ff98698a1f80b5c6715969644f519","timestamp":"2016-10-04T19:13:23Z","block_id":"181c11b24c7dbdd5ce5e2b9da1b665878a80712dbfd1796613e66305da49ca7c","block_height":2,"position":0,"reference_data":{},"is_local":"yes","inputs":[{"action":"issue","asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":5,"issuance_program":"027b7d75766baa205fd08ca9e18b180c7da3ace70e890cba8c7014c7da5c9ed78e3d9e253cccec8f5151ad696c00c0","reference_data":{},"is_local":"yes"}],"outputs":[{"action":"control","purpose":"receive","position":0,"asset_id":"1811eb7d8aebbfc39ea14a2da0ae840e9b447952d6e205756ea5c7ad028bcc97","asset_alias":"t","asset_definition":{},"asset_tags":{},"asset_is_local":"yes","amount":5,"account_id":"acc0KNY9W8QG0802","account_alias":"t","account_tags":null,"control_program":"766baa20107c8767129a4ae5325371946e44f4ae76448452f722768048bd2d5cf12fc1595151ad696c00c0","reference_data":{},"is_local":"yes"}]}],"next":{"page_size":0,"timeout":0,"after":"2:0-1","end_time":1475613062958,"type":""},"last_page":true}`)
)

type noOpWriter struct{ header http.Header }

func (n noOpWriter) Header() http.Header {
	return n.header
}

func (n noOpWriter) Write(d []byte) (int, error) {
	return len(d), nil
}

func (n noOpWriter) WriteHeader(int) {}

func BenchmarkGzipSmall(b *testing.B) {
	r, _ := http.NewRequest("GET", "/foo", nil) // #nosec
	r.Header.Set("accept-encoding", "gzip")
	h := Handler{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(small)
	})}
	w := noOpWriter{header: http.Header{}}

	for i := 0; i < b.N; i++ {
		h.ServeHTTP(&w, r)
	}
	b.SetBytes(int64(len(small)))
}

func BenchmarkGzipMedium(b *testing.B) {
	r, _ := http.NewRequest("GET", "/foo", nil)
	r.Header.Set("accept-encoding", "gzip")
	h := Handler{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(medium)
	})}
	w := noOpWriter{header: http.Header{}}

	for i := 0; i < b.N; i++ {
		h.ServeHTTP(&w, r)
	}
	b.SetBytes(int64(len(medium)))
}

func BenchmarkGzipLarge(b *testing.B) {
	r, _ := http.NewRequest("GET", "/foo", nil)
	r.Header.Set("accept-encoding", "gzip")
	h := Handler{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(large)
	})}
	w := noOpWriter{header: http.Header{}}

	for i := 0; i < b.N; i++ {
		h.ServeHTTP(&w, r)
	}
	b.SetBytes(int64(len(large)))
}

func TestGzip(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/foo", nil)
	r.Header.Set("accept-encoding", "gzip")
	h := Handler{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello, world")
	})}
	h.ServeHTTP(w, r)
	if s := w.HeaderMap.Get("content-encoding"); s != "gzip" {
		t.Errorf(`w.HeaderMap.Get("content-encoding") = %s want gzip`, s)
	}
}

func TestNoGzip(t *testing.T) {
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/foo", nil)
	h := Handler{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello, world")
	})}
	h.ServeHTTP(w, r)
	if w.HeaderMap.Get("content-encoding") == "gzip" {
		t.Error("unexpected gzip")
	}
}
