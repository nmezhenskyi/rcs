//go:build !rmhttp

package httpsrv

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	server := NewServer(nil)
	if server == nil {
		t.Error("Expected pointer to initialized Server, got nil instead")
		return
	}
	if server.server == nil {
		t.Error("Server.server has not been initialized")
	}
	if server.router == nil {
		t.Error("Server.router has not been initialized")
	}
	if server.cache == nil {
		t.Error("Server.cache has not been initialized")
	}
}

func TestListenAndServe(t *testing.T) {
	// Make sure Server implements http.Handler
	var _ http.Handler = (*Server)(nil)
	srv := NewServer(nil)
	done := make(chan error)
	go func(done chan<- error) {
		done <- srv.ListenAndServe("localhost:6123")
	}(done)
	time.Sleep(500 * time.Millisecond)
	srv.Close()
	err := <-done
	if err != nil {
		t.Errorf("ListenAndServe failed with: %v", err)
	}
}

func TestShutdown(t *testing.T) {
	srv := NewServer(nil)
	done := make(chan error)
	go func(done chan<- error) {
		done <- srv.ListenAndServe("localhost:6123")
	}(done)
	time.Sleep(500 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	err = <-done
	if err != nil {
		t.Errorf("ListenAndServe failed with: %v", err)
	}

	_, err = http.Get("http://localhost:6123/PING")
	if err == nil {
		t.Error("Expected GET /PING to fail after shutdown, but the request was processed")
	}
}

func TestSet(t *testing.T) {
	server := NewServer(nil)

	testCases := []struct {
		name         string
		key          string
		value        []byte
		expectedCode int
	}{
		{
			name:         "Empty key, no value",
			key:          "",
			value:        nil,
			expectedCode: http.StatusNotFound,
		},
		{
			name:         "Valid key, no value",
			key:          "key1",
			value:        nil,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Valid key, valid value",
			key:          "key1",
			value:        []byte("10"),
			expectedCode: http.StatusOK,
		},
	}

	type request struct {
		Value string `json:"value"`
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("/SET/%s", tc.key)
			reqData := request{base64.StdEncoding.EncodeToString(tc.value)}
			byteData, err := json.Marshal(reqData)
			if err != nil {
				t.Errorf("Failed to encode data into JSON: %v", err)
			}
			res, err := sendRequest("PUT", url, bytes.NewReader(byteData), server)
			if err != nil {
				t.Errorf("Failed to send request: %v", err)
			}
			if code := res.Result().StatusCode; code != tc.expectedCode {
				t.Errorf("Expected response status code %d, got %d instead", tc.expectedCode, code)
			}
		})
	}
}

func TestGet(t *testing.T) {
	server := NewServer(nil)

	testCases := []struct {
		name          string
		key           string
		ok            bool
		expectedValue []byte
		expectedCode  int
	}{
		{
			name:         "Invalid key, no value",
			key:          "",
			expectedCode: http.StatusNotFound,
		},
		{
			name:          "Valid key, no value",
			key:           "key1",
			ok:            false,
			expectedValue: nil,
			expectedCode:  http.StatusOK,
		},
		{
			name:          "Valid key, valid value",
			key:           "key1",
			ok:            true,
			expectedValue: []byte("10"),
			expectedCode:  http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.key != "" {
				server.cache.Set(tc.key, tc.expectedValue)
			}

			url := fmt.Sprintf("/GET/%s", tc.key)
			res, err := sendRequest("GET", url, nil, server)
			if err != nil {
				t.Errorf("Failed to send request: %v", err)
			}
			if code := res.Result().StatusCode; code != tc.expectedCode {
				t.Errorf("Expected response status code %d, got %d instead", tc.expectedCode, code)
			}

			resData := httpResponse{}
			json.NewDecoder(res.Body).Decode(&resData)
			val, ok := resData.Value.(string)
			if tc.expectedValue != nil && !ok {
				t.Error("Not ok")
			}
			if !bytes.Equal([]byte(val), tc.expectedValue) {
				t.Errorf("Expected value %v, got %v instead", tc.expectedValue, []byte(val))
			}
		})
	}
}

func TestDelete(t *testing.T) {
	server := NewServer(nil)
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))
	lengthBefore := server.cache.Length()

	url := fmt.Sprintf("/DELETE/%s", "key1")
	res, err := sendRequest("DELETE", url, nil, server)
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}
	if code := res.Result().StatusCode; code != http.StatusOK {
		t.Errorf("Expected response status code %d, got %d instead", http.StatusOK, code)
	}
	if server.cache.Length() != lengthBefore-1 {
		t.Errorf("Cache length has not changed")
	}
}

func TestPurge(t *testing.T) {
	server := NewServer(nil)
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))

	res, err := sendRequest("DELETE", "/PURGE", nil, server)
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}
	if code := res.Result().StatusCode; code != http.StatusOK {
		t.Errorf("Expected response status code %d, got %d instead", http.StatusOK, code)
	}
	if server.cache.Length() != 0 {
		t.Errorf("Cache is not empty")
	}
}

func TestLength(t *testing.T) {
	server := NewServer(nil)
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))
	server.cache.Set("key4", []byte("40"))
	server.cache.Set("key5", []byte("50"))

	res, err := sendRequest("GET", "/LENGTH", nil, server)
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}
	if code := res.Result().StatusCode; code != http.StatusOK {
		t.Errorf("Expected response status code %d, got %d instead", http.StatusOK, code)
	}

	resData := httpResponse{}
	json.NewDecoder(res.Body).Decode(&resData)
	val, ok := resData.Value.(float64)
	if !ok {
		t.Error("Returned value is not a number")
	}
	actualLength := server.cache.Length()
	if int(val) != actualLength {
		t.Errorf("Expected length %d, got %d instead", actualLength, int(val))
	}
}

func TestKeys(t *testing.T) {
	server := NewServer(nil)
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))
	server.cache.Set("key4", []byte("40"))
	server.cache.Set("key5", []byte("50"))

	res, err := sendRequest("GET", "/KEYS", nil, server)
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}
	if code := res.Result().StatusCode; code != http.StatusOK {
		t.Errorf("Expected response status code %d, got %d instead", http.StatusOK, code)
	}

	expectedKeys := server.cache.Keys()
	resData := httpResponse{}
	json.NewDecoder(res.Body).Decode(&resData)
	val, ok := resData.Value.([]any)
	if !ok {
		t.Errorf("Returned value is not an array")
	}
	keys := make([]string, len(val))
	for i := range val {
		keys[i], ok = val[i].(string)
		if !ok {
			t.Errorf("Returned key is not a string: %v", val[i])
		}
	}
	receivedKeys := strings.Join(keys, ",")
	for i := range expectedKeys {
		if !strings.Contains(receivedKeys, expectedKeys[i]) {
			t.Errorf("Key \"%s\" not found", expectedKeys[i])
		}
	}
}

func TestPing(t *testing.T) {
	server := NewServer(nil)
	res, err := sendRequest("GET", "/PING", nil, server)
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}
	if code := res.Result().StatusCode; code != http.StatusOK {
		t.Errorf("Expected response status code %d, got %d instead", http.StatusOK, code)
	}
}

func sendRequest(
	method, url string,
	body io.Reader,
	server *Server,
) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	return rr, nil
}
