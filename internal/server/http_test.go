package server

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := fmt.Sprintf("/SET/%s", tc.key)
			byteData := []byte(base64.StdEncoding.EncodeToString(tc.value))
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

}

func TestDeleteHTTP(t *testing.T) {

}

func TestFlushHTTP(t *testing.T) {

}

func TestLengthHTTP(t *testing.T) {

}

func TestPingHTTP(t *testing.T) {
	server := NewHTTPServer()
	res, err := sendRequestHTTP("GET", "/PING", nil, server)
	if err != nil {
		t.Errorf("Failed on GET \"/PING\" request: %v", err)
	}
	if code := res.Result().StatusCode; code != 200 {
		t.Errorf("Expected response status code 200, got %v", code)
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
