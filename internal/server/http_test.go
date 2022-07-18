package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewHTTPServer(t *testing.T) {
	server := NewHTTPServer()
	if server == nil {
		t.Error("Expected pointer to initialized HTTPServer, got nil instead")
	}
}

func TestSetHTTP(t *testing.T) {
	server := NewHTTPServer()

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
			res, err := sendRequestHTTP("PUT", url, bytes.NewReader(byteData), server)
			if err != nil {
				t.Errorf("Failed to send request: %v", err)
			}
			if code := res.Result().StatusCode; code != tc.expectedCode {
				t.Errorf("Expected response status code %d, got %d", tc.expectedCode, code)
			}
		})
	}
}

func TestGetHTTP(t *testing.T) {
	server := NewHTTPServer()

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
			res, err := sendRequestHTTP("GET", url, nil, server)
			if err != nil {
				t.Errorf("Failed to send request: %v", err)
			}
			if code := res.Result().StatusCode; code != tc.expectedCode {
				t.Errorf("Expected response status code %d, got %d", tc.expectedCode, code)
			}

			resData := httpResponse{}
			json.NewDecoder(res.Body).Decode(&resData)
			val, ok := resData.Value.(string)
			if tc.expectedValue != nil && !ok {
				t.Error("Not ok")
			}
			if bytes.Compare([]byte(val), tc.expectedValue) != 0 {
				t.Errorf("Expected value %v, instead got %v", tc.expectedValue, []byte(val))
			}
		})
	}
}

func TestDeleteHTTP(t *testing.T) {
	server := NewHTTPServer()
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))
	lengthBefore := server.cache.Length()

	url := fmt.Sprintf("/DELETE/%s", "key1")
	res, err := sendRequestHTTP("DELETE", url, nil, server)
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}
	if code := res.Result().StatusCode; code != http.StatusOK {
		t.Errorf("Expected response status code %d, got %d", http.StatusOK, code)
	}
	if server.cache.Length() != lengthBefore-1 {
		t.Errorf("Cache length has not changed")
	}
}

func TestFlushHTTP(t *testing.T) {
	server := NewHTTPServer()
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))

	res, err := sendRequestHTTP("DELETE", "/FLUSH", nil, server)
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}
	if code := res.Result().StatusCode; code != http.StatusOK {
		t.Errorf("Expected response status code %d, got %d", http.StatusOK, code)
	}
	if server.cache.Length() != 0 {
		t.Errorf("Cache is not empty")
	}
}

func TestLengthHTTP(t *testing.T) {
	server := NewHTTPServer()
	server.cache.Set("key1", []byte("10"))
	server.cache.Set("key2", []byte("20"))
	server.cache.Set("key3", []byte("30"))
	server.cache.Set("key4", []byte("40"))
	server.cache.Set("key5", []byte("50"))

	res, err := sendRequestHTTP("GET", "/LENGTH", nil, server)
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}
	if code := res.Result().StatusCode; code != http.StatusOK {
		t.Errorf("Expected response status code %d, got %d", http.StatusOK, code)
	}

	resData := httpResponse{}
	json.NewDecoder(res.Body).Decode(&resData)
	val, ok := resData.Value.(float64)
	if !ok {
		t.Error("Returned value is not a number")
	}
	actualLength := server.cache.Length()
	if int(val) != actualLength {
		t.Errorf("Expected length %d, got %d", actualLength, int(val))
	}
}

func TestPingHTTP(t *testing.T) {
	server := NewHTTPServer()
	res, err := sendRequestHTTP("GET", "/PING", nil, server)
	if err != nil {
		t.Errorf("Failed to send request: %v", err)
	}
	if code := res.Result().StatusCode; code != http.StatusOK {
		t.Errorf("Expected response status code %d, got %d", http.StatusOK, code)
	}
}

func sendRequestHTTP(
	method, url string,
	body io.Reader,
	server *HTTPServer,
) (*httptest.ResponseRecorder, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	rr := httptest.NewRecorder()
	server.ServeHTTP(rr, req)

	return rr, nil
}
