package nativesrv

import (
	"bytes"
	"net"
	"testing"
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

	testCases := []struct {
		name             string
		key              []byte
		value            []byte
		expectedResponse response
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
		})
	}
}
