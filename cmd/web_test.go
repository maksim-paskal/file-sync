package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouting_Sync(t *testing.T) {
	ctx := context.Background()
	web := newWeb()

	srv := httptest.NewServer(web.getHTTPRouter())
	defer srv.Close()

	syncURL := fmt.Sprintf("%s/api/queue?value=put:tests/test.txt", srv.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, syncURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "ok" {
		t.Errorf("text %s not OK", string(body))
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("status %d not OK", res.StatusCode)
	}
}
