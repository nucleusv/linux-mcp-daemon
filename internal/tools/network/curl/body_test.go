package curl

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyIsSentAndHTMLNotEscaped(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.Write([]byte("<html>ok</html>"))
	}))
	defer srv.Close()

	args, _ := json.Marshal(map[string]interface{}{"url": srv.URL, "method": "POST", "body": `{"a":1}`})
	out, err := Curl(args)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"a":1}` {
		t.Errorf("server received body %q - the schema's \"body\" param was dropped", got)
	}
	if !strings.Contains(out, "<html>ok</html>") {
		t.Errorf("response body HTML-escaped: %s", out)
	}
}
