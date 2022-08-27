package nativesrv

import (
	"bytes"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	server := NewServer(nil)
	if server == nil {
		t.Error("Expected pointer to initialized Server, got nil instead")
	}
	if server.cache == nil {
		t.Error("Server.cache has not been initialized")
	}
}

func TestSet(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5000"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	defer server.Close()

	testCases := []struct {
		name             string
		key              []byte
		value            []byte
		expectedResponse response
		verifyInCache    bool
	}{
		{
			name:  "Nil key, nil value",
			key:   nil,
			value: nil,
			expectedResponse: response{
				command: []byte("SET"),
				message: []byte("Key is missing"),
				ok:      false,
				key:     nil,
				value:   nil,
			},
		},
		{
			name:  "Empty key, empty value",
			key:   []byte(""),
			value: []byte(""),
			expectedResponse: response{
				command: []byte("SET"),
				message: []byte("Key is missing"),
				ok:      false,
				key:     nil,
				value:   nil,
			},
		},
		{
			name:  "Valid key, empty value",
			key:   []byte("key1"),
			value: []byte(""),
			expectedResponse: response{
				command: []byte("SET"),
				message: []byte("Value is missing"),
				ok:      false,
				key:     []byte("key1"),
				value:   nil,
			},
		},
		{
			name:  "Empty key, valid value",
			key:   []byte(""),
			value: []byte("val1"),
			expectedResponse: response{
				command: []byte("SET"),
				message: []byte("Key is missing"),
				ok:      false,
				key:     nil,
				value:   nil,
			},
		},
		{
			name:  "Valid key, valid value",
			key:   []byte("key1"),
			value: []byte("val1"),
			expectedResponse: response{
				command: []byte("SET"),
				message: nil,
				ok:      true,
				key:     []byte("key1"),
				value:   nil,
			},
			verifyInCache: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := request{
				command: []byte("SET"),
				key:     tc.key,
				value:   tc.value,
			}
			conn, err := net.Dial("tcp", serverAddr)
			if err != nil {
				t.Errorf("Failed to connect to the server: %v", err)
			}
			req.write(conn)

			respBuf := [1024]byte{}
			n, err := conn.Read(respBuf[:])
			if err != nil {
				t.Errorf("Error while reading from server")
			}
			resp, err := parseResponse(respBuf[:n])
			if err != nil {
				t.Logf("Response buffer:\n%s", string(respBuf[:n]))
				t.Logf("Error while parsing response: %v", err)
			}

			if resp.ok != tc.expectedResponse.ok {
				t.Errorf("Expected ok to be \"%v\", got \"%v\" instead",
					tc.expectedResponse.ok, resp.ok)
			}
			if bytes.Compare(resp.command, tc.expectedResponse.command) != 0 {
				t.Errorf("Expected command to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.command), string(resp.command))
			}
			if bytes.Compare(resp.message, tc.expectedResponse.message) != 0 {
				t.Errorf("Expected message to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.message), string(resp.message))
			}
			if bytes.Compare(resp.key, tc.expectedResponse.key) != 0 {
				t.Errorf("Expected key to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.key), string(resp.key))
			}
			if bytes.Compare(resp.value, tc.expectedResponse.value) != 0 {
				t.Errorf("Expected value to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.value), string(resp.value))
			}
			if tc.verifyInCache {
				key := string(req.key)
				valInCache, ok := server.cache.Get(key)
				if !ok {
					t.Error("Value is missing in Server.cache")
				}
				if bytes.Compare(req.value, valInCache) != 0 {
					t.Errorf("Expected value in Server.cache to be \"%s\", got \"%s\" instead",
						string(req.value), string(valInCache))
				}
			}
		})
	}
}

func TestGet(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5000"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	defer server.Close()

	testCases := []struct {
		name             string
		key              []byte
		payloadValue     []byte
		expectedResponse response
	}{
		{
			name: "Nil key",
			key:  nil,
			expectedResponse: response{
				command: []byte("GET"),
				message: []byte("Key is missing"),
				ok:      false,
				key:     nil,
				value:   nil,
			},
		},
		{
			name: "Empty key",
			key:  []byte(""),
			expectedResponse: response{
				command: []byte("GET"),
				message: []byte("Key is missing"),
				ok:      false,
				key:     nil,
				value:   nil,
			},
		},
		{
			name:         "Valid key, unnecessary value",
			key:          []byte("key1"),
			payloadValue: []byte("some_value"),
			expectedResponse: response{
				command: []byte("GET"),
				message: []byte("Received unexpected value"),
				ok:      false,
				key:     []byte("key1"),
				value:   nil,
			},
		},
		{
			name: "Valid key, valid value",
			key:  []byte("key1"),
			expectedResponse: response{
				command: []byte("GET"),
				message: nil,
				ok:      true,
				key:     []byte("key1"),
				value:   []byte("some_value"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.key) != 0 && tc.expectedResponse.value != nil {
				server.cache.Set(string(tc.key), tc.expectedResponse.value)
			}
			req := request{
				command: []byte("GET"),
				key:     tc.key,
				value:   tc.payloadValue,
			}
			conn, err := net.Dial("tcp", serverAddr)
			if err != nil {
				t.Errorf("Failed to connect to the server: %v", err)
			}
			req.write(conn)

			respBuf := [1024]byte{}
			n, err := conn.Read(respBuf[:])
			if err != nil {
				t.Errorf("Error while reading from server")
			}
			resp, err := parseResponse(respBuf[:n])
			if err != nil {
				t.Logf("Response buffer:\n%s", string(respBuf[:n]))
				t.Logf("Error while parsing response: %v", err)
			}

			if resp.ok != tc.expectedResponse.ok {
				t.Errorf("Expected ok to be \"%v\", got \"%v\" instead",
					tc.expectedResponse.ok, resp.ok)
			}
			if bytes.Compare(resp.command, tc.expectedResponse.command) != 0 {
				t.Errorf("Expected command to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.command), string(resp.command))
			}
			if bytes.Compare(resp.message, tc.expectedResponse.message) != 0 {
				t.Errorf("Expected message to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.message), string(resp.message))
			}
			if bytes.Compare(resp.key, tc.expectedResponse.key) != 0 {
				t.Errorf("Expected key to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.key), string(resp.key))
			}
			if bytes.Compare(resp.value, tc.expectedResponse.value) != 0 {
				t.Errorf("Expected value to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.value), string(resp.value))
			}
		})
	}
}

func TestDelete(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5000"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	defer server.Close()

	testCases := []struct {
		name             string
		key              []byte
		cacheValue       []byte
		expectedResponse response
	}{
		{
			name:       "Nil key",
			key:        nil,
			cacheValue: []byte("some_value"),
			expectedResponse: response{
				command: []byte("DELETE"),
				message: []byte("Key is missing"),
				ok:      false,
				key:     nil,
				value:   nil,
			},
		},
		{
			name: "Empty key",
			key:  []byte(""),
			expectedResponse: response{
				command: []byte("DELETE"),
				message: []byte("Key is missing"),
				ok:      false,
				key:     nil,
				value:   nil,
			},
		},
		{
			name: "Valid key",
			key:  []byte("key1"),
			expectedResponse: response{
				command: []byte("DELETE"),
				message: nil,
				ok:      true,
				key:     []byte("key1"),
				value:   nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.key) != 0 && tc.expectedResponse.value != nil {
				server.cache.Set(string(tc.key), tc.cacheValue)
			}
			req := request{
				command: []byte("DELETE"),
				key:     tc.key,
			}
			conn, err := net.Dial("tcp", serverAddr)
			if err != nil {
				t.Errorf("Failed to connect to the server: %v", err)
			}
			req.write(conn)

			respBuf := [1024]byte{}
			n, err := conn.Read(respBuf[:])
			if err != nil {
				t.Errorf("Error while reading from server")
			}
			resp, err := parseResponse(respBuf[:n])
			if err != nil {
				t.Logf("Response buffer:\n%s", string(respBuf[:n]))
				t.Logf("Error while parsing response: %v", err)
			}

			if resp.ok != tc.expectedResponse.ok {
				t.Errorf("Expected ok to be \"%v\", got \"%v\" instead",
					tc.expectedResponse.ok, resp.ok)
			}
			if bytes.Compare(resp.command, tc.expectedResponse.command) != 0 {
				t.Errorf("Expected command to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.command), string(resp.command))
			}
			if bytes.Compare(resp.message, tc.expectedResponse.message) != 0 {
				t.Errorf("Expected message to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.message), string(resp.message))
			}
			if bytes.Compare(resp.key, tc.expectedResponse.key) != 0 {
				t.Errorf("Expected key to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.key), string(resp.key))
			}
			if bytes.Compare(resp.value, tc.expectedResponse.value) != 0 {
				t.Errorf("Expected value to be \"%s\", got \"%s\" instead",
					string(tc.expectedResponse.value), string(resp.value))
			}
			if _, ok := server.cache.Get(string(resp.key)); ok {
				t.Errorf("Value was not removed from Server.cache")
			}
		})
	}
}

func TestPurge(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5000"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	defer server.Close()

	expectedResponse := response{
		command: []byte("PURGE"),
		message: nil,
		ok:      true,
		key:     nil,
		value:   nil,
	}

	server.cache.Set("key1", []byte("value1"))
	server.cache.Set("key2", []byte("value2"))
	server.cache.Set("key3", []byte("value3"))

	req := request{command: []byte("PURGE")}
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Errorf("Failed to connect to the server: %v", err)
	}
	req.write(conn)

	respBuf := [1024]byte{}
	n, err := conn.Read(respBuf[:])
	if err != nil {
		t.Errorf("Error while reading from server")
	}
	resp, err := parseResponse(respBuf[:n])
	if err != nil {
		t.Logf("Response buffer:\n%s", string(respBuf[:n]))
		t.Logf("Error while parsing response: %v", err)
	}

	if resp.ok != expectedResponse.ok {
		t.Errorf("Expected ok to be \"%v\", got \"%v\" instead",
			expectedResponse.ok, resp.ok)
	}
	if bytes.Compare(resp.command, expectedResponse.command) != 0 {
		t.Errorf("Expected command to be \"%s\", got \"%s\" instead",
			string(expectedResponse.command), string(resp.command))
	}
	if bytes.Compare(resp.message, expectedResponse.message) != 0 {
		t.Errorf("Expected message to be \"%s\", got \"%s\" instead",
			string(expectedResponse.message), string(resp.message))
	}
	if bytes.Compare(resp.key, expectedResponse.key) != 0 {
		t.Errorf("Expected key to be \"%s\", got \"%s\" instead",
			string(expectedResponse.key), string(resp.key))
	}
	if bytes.Compare(resp.value, expectedResponse.value) != 0 {
		t.Errorf("Expected value to be \"%s\", got \"%s\" instead",
			string(expectedResponse.value), string(resp.value))
	}
	if server.cache.Length() != 0 {
		t.Error("Server.cache is not empty")
	}
}

func TestLength(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5000"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	defer server.Close()

	expectedResponse := response{
		command: []byte("LENGTH"),
		message: nil,
		ok:      true,
		key:     nil,
		value:   []byte("5"),
	}

	server.cache.Set("key1", []byte("value1"))
	server.cache.Set("key2", []byte("value2"))
	server.cache.Set("key3", []byte("value3"))
	server.cache.Set("key4", []byte("value4"))
	server.cache.Set("key5", []byte("value5"))

	req := request{command: []byte("LENGTH")}
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Errorf("Failed to connect to the server: %v", err)
	}
	req.write(conn)

	respBuf := [1024]byte{}
	n, err := conn.Read(respBuf[:])
	if err != nil {
		t.Errorf("Error while reading from server")
	}
	resp, err := parseResponse(respBuf[:n])
	if err != nil {
		t.Logf("Response buffer:\n%s", string(respBuf[:n]))
		t.Logf("Error while parsing response: %v", err)
	}

	if resp.ok != expectedResponse.ok {
		t.Errorf("Expected ok to be \"%v\", got \"%v\" instead",
			expectedResponse.ok, resp.ok)
	}
	if bytes.Compare(resp.command, expectedResponse.command) != 0 {
		t.Errorf("Expected command to be \"%s\", got \"%s\" instead",
			string(expectedResponse.command), string(resp.command))
	}
	if bytes.Compare(resp.message, expectedResponse.message) != 0 {
		t.Errorf("Expected message to be \"%s\", got \"%s\" instead",
			string(expectedResponse.message), string(resp.message))
	}
	if bytes.Compare(resp.key, expectedResponse.key) != 0 {
		t.Errorf("Expected key to be \"%s\", got \"%s\" instead",
			string(expectedResponse.key), string(resp.key))
	}
	if bytes.Compare(resp.value, expectedResponse.value) != 0 {
		t.Errorf("Expected value to be \"%s\", got \"%s\" instead",
			string(expectedResponse.value), string(resp.value))
	}
	cacheLen := strconv.Itoa(server.cache.Length())
	if cacheLen != string(expectedResponse.value) {
		t.Errorf("Expected Server.cache.Length() to be %s, got %s instead",
			string(expectedResponse.value), cacheLen)
	}
}

func TestKeys(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5000"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	defer server.Close()

	expectedResponse := response{
		command: []byte("KEYS"),
		message: nil,
		ok:      true,
		key:     nil,
		value:   []byte("key1,key2,key3,key4,key5"),
	}

	server.cache.Set("key1", []byte("value1"))
	server.cache.Set("key2", []byte("value2"))
	server.cache.Set("key3", []byte("value3"))
	server.cache.Set("key4", []byte("value4"))
	server.cache.Set("key5", []byte("value5"))

	req := request{command: []byte("KEYS")}
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Errorf("Failed to connect to the server: %v", err)
	}
	req.write(conn)

	respBuf := [1024]byte{}
	n, err := conn.Read(respBuf[:])
	if err != nil {
		t.Errorf("Error while reading from server")
	}
	resp, err := parseResponse(respBuf[:n])
	if err != nil {
		t.Logf("Response buffer:\n%s", string(respBuf[:n]))
		t.Logf("Error while parsing response: %v", err)
	}

	if resp.ok != expectedResponse.ok {
		t.Errorf("Expected ok to be \"%v\", got \"%v\" instead",
			expectedResponse.ok, resp.ok)
	}
	if bytes.Compare(resp.command, expectedResponse.command) != 0 {
		t.Errorf("Expected command to be \"%s\", got \"%s\" instead",
			string(expectedResponse.command), string(resp.command))
	}
	if bytes.Compare(resp.message, expectedResponse.message) != 0 {
		t.Errorf("Expected message to be \"%s\", got \"%s\" instead",
			string(expectedResponse.message), string(resp.message))
	}
	if bytes.Compare(resp.key, expectedResponse.key) != 0 {
		t.Errorf("Expected key to be \"%s\", got \"%s\" instead",
			string(expectedResponse.key), string(resp.key))
	}

	expectedKeys := strings.Split(string(expectedResponse.value), ",")
	actualKeys := strings.Split(string(resp.value), ",")
	if len(expectedKeys) != len(actualKeys) {
		t.Errorf("Expected to receive %d keys, got %d instead", len(expectedKeys), len(actualKeys))
	}
	for i := range expectedKeys {
		if !strings.Contains(string(resp.value), expectedKeys[i]) {
			t.Errorf("Key \"%s\" not found", expectedKeys[i])
		}
	}
}

func TestPing(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5000"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	defer server.Close()

	expectedResponse := response{
		command: []byte("PING"),
		message: []byte("PONG"),
		ok:      true,
		key:     nil,
		value:   nil,
	}

	req := request{command: []byte("PING")}
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Errorf("Failed to connect to the server: %v", err)
	}
	req.write(conn)

	respBuf := [1024]byte{}
	n, err := conn.Read(respBuf[:])
	if err != nil {
		t.Errorf("Error while reading from server")
	}
	resp, err := parseResponse(respBuf[:n])
	if err != nil {
		t.Logf("Response buffer:\n%s", string(respBuf[:n]))
		t.Logf("Error while parsing response: %v", err)
	}

	if resp.ok != expectedResponse.ok {
		t.Errorf("Expected ok to be \"%v\", got \"%v\" instead",
			expectedResponse.ok, resp.ok)
	}
	if bytes.Compare(resp.command, expectedResponse.command) != 0 {
		t.Errorf("Expected command to be \"%s\", got \"%s\" instead",
			string(expectedResponse.command), string(resp.command))
	}
	if bytes.Compare(resp.message, expectedResponse.message) != 0 {
		t.Errorf("Expected message to be \"%s\", got \"%s\" instead",
			string(expectedResponse.message), string(resp.message))
	}
	if bytes.Compare(resp.key, expectedResponse.key) != 0 {
		t.Errorf("Expected key to be \"%s\", got \"%s\" instead",
			string(expectedResponse.key), string(resp.key))
	}
	if bytes.Compare(resp.value, expectedResponse.value) != 0 {
		t.Errorf("Expected value to be \"%s\", got \"%s\" instead",
			string(expectedResponse.value), string(resp.value))
	}
}

func TestClose(t *testing.T) {
	server := NewServer(nil)
	serverAddr := "localhost:5000"
	go func() {
		if err := server.ListenAndServe(serverAddr); err != nil {
			t.Errorf("Server failed: %v", err)
		}
	}()
	defer server.Close()

	expectedResponse := response{
		command: []byte("CLOSE"),
		message: nil,
		ok:      true,
		key:     nil,
		value:   nil,
	}

	req := request{command: []byte("CLOSE")}
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Errorf("Failed to connect to the server: %v", err)
	}
	req.write(conn)

	respBuf := [1024]byte{}
	n, err := conn.Read(respBuf[:])
	if err != nil {
		t.Errorf("Error while reading from server")
	}
	resp, err := parseResponse(respBuf[:n])
	if err != nil {
		t.Logf("Response buffer:\n%s", string(respBuf[:n]))
		t.Logf("Error while parsing response: %v", err)
	}

	if resp.ok != expectedResponse.ok {
		t.Errorf("Expected ok to be \"%v\", got \"%v\" instead",
			expectedResponse.ok, resp.ok)
	}
	if bytes.Compare(resp.command, expectedResponse.command) != 0 {
		t.Errorf("Expected command to be \"%s\", got \"%s\" instead",
			string(expectedResponse.command), string(resp.command))
	}
	if bytes.Compare(resp.message, expectedResponse.message) != 0 {
		t.Errorf("Expected message to be \"%s\", got \"%s\" instead",
			string(expectedResponse.message), string(resp.message))
	}
	if bytes.Compare(resp.key, expectedResponse.key) != 0 {
		t.Errorf("Expected key to be \"%s\", got \"%s\" instead",
			string(expectedResponse.key), string(resp.key))
	}
	if bytes.Compare(resp.value, expectedResponse.value) != 0 {
		t.Errorf("Expected value to be \"%s\", got \"%s\" instead",
			string(expectedResponse.value), string(resp.value))
	}

	one := make([]byte, 1)
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	_, err = conn.Read(one)
	if err == nil {
		t.Error("Expected read from server to fail with EOF after CLOSE but operation succeeded")
	}
	if err != io.EOF {
		t.Errorf("Expected read from server to fail with EOF after CLOSE but got different error: %v", err)
	}
}
