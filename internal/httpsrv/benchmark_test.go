//go:build !rmhttp

package httpsrv

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func BenchmarkSet(b *testing.B) {
	type request struct {
		Value string `json:"value"`
	}

	server := NewServer(nil)
	serverAddr := "localhost:5002"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			b.Errorf("Server failed: %v", err)
		}
	}()
	defer server.Close()

	url := fmt.Sprintf("http://%s/SET/%s", serverAddr, "apollo")
	value := []byte("Apollo is one of the Olympian deities in classical Greek and Roman religion and Greek and Roman mythology. (From Wikipedia, the free encyclopedia)")
	reqData := request{base64.StdEncoding.EncodeToString(value)}
	byteData, err := json.Marshal(reqData)
	if err != nil {
		b.Errorf("Failed to encode data into JSON: %v", err)
	}
	rdr := bytes.NewReader(byteData)
	req, err := http.NewRequest("PUT", url, rdr)
	if err != nil {
		b.Errorf("Failed to init request: %v", err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		rdr.Reset(byteData)
		_, err = http.DefaultClient.Do(req)
		if err != nil {
			b.Errorf("Failed to send request: %v", err)
		}
	}
}
