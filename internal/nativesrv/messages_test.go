package nativesrv

import (
	"bytes"
	"testing"
)

func TestParseRequest(t *testing.T) {
	testCases := []struct {
		name        string
		msg         []byte
		expectedReq request
		expectedErr error
	}{
		{
			name:        "Nil request message",
			msg:         nil,
			expectedReq: request{},
			expectedErr: ErrMalformedRequest,
		},
		{
			name:        "Empty request message",
			msg:         nil,
			expectedReq: request{},
			expectedErr: ErrMalformedRequest,
		},
		{
			name:        "Invalid protocol",
			msg:         []byte("ABCD SET\r\n"),
			expectedReq: request{},
			expectedErr: ErrUnknownProtocol,
		},
		{
			name: "Valid SET request",
			msg:  []byte("RCSP/1.0 SET\r\nKEY: key1\r\nVALUE: 10\r\n"),
			expectedReq: request{
				command: []byte("SET"),
				key:     []byte("key1"),
				value:   []byte("10"),
			},
			expectedErr: nil,
		},
		{
			name: "Valid GET request",
			msg:  []byte("RCSP/1.0 GET\r\nKEY: key1\r\n"),
			expectedReq: request{
				command: []byte("GET"),
				key:     []byte("key1"),
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Valid DELETE request",
			msg:  []byte("RCSP/1.0 DELETE\r\nKEY: key1\r\n"),
			expectedReq: request{
				command: []byte("DELETE"),
				key:     []byte("key1"),
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Valid PURGE request",
			msg:  []byte("RCSP/1.0 PURGE\r\n"),
			expectedReq: request{
				command: []byte("PURGE"),
				key:     nil,
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Valid LENGTH request",
			msg:  []byte("RCSP/1.0 LENGTH\r\n"),
			expectedReq: request{
				command: []byte("LENGTH"),
				key:     nil,
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Valid KEYS request",
			msg:  []byte("RCSP/1.0 KEYS\r\n"),
			expectedReq: request{
				command: []byte("KEYS"),
				key:     nil,
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Valid PING request",
			msg:  []byte("RCSP/1.0 PING\r\n"),
			expectedReq: request{
				command: []byte("PING"),
				key:     nil,
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Valid CLOSE request",
			msg:  []byte("RCSP/1.0 CLOSE\r\n"),
			expectedReq: request{
				command: []byte("CLOSE"),
				key:     nil,
				value:   nil,
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := parseRequest(tc.msg)
			if err != tc.expectedErr {
				t.Errorf("Expected error value \"%v\", got \"%v\" instead",
					tc.expectedErr, err)
			}
			if bytes.Compare(req.command, tc.expectedReq.command) != 0 {
				t.Errorf("Expected command \"%s\", got \"%s\" instead",
					string(tc.expectedReq.command), string(req.command))
			}
			if bytes.Compare(req.key, tc.expectedReq.key) != 0 {
				t.Errorf("Expected key \"%s\", got \"%s\" instead",
					string(tc.expectedReq.key), string(req.key))
			}
			if bytes.Compare(req.value, tc.expectedReq.value) != 0 {
				t.Errorf("Expected value \"%s\", got \"%s\" instead",
					string(tc.expectedReq.value), string(req.value))
			}
		})
	}
}

func TestParseResponse(t *testing.T) {
	testCases := []struct {
		name         string
		msg          []byte
		expectedResp response
		expectedErr  error
	}{
		{
			name:         "Nil response message",
			msg:          nil,
			expectedResp: response{},
			expectedErr:  ErrMalformedResponse,
		},
		{
			name:         "Empty response message",
			msg:          nil,
			expectedResp: response{},
			expectedErr:  ErrMalformedResponse,
		},
		{
			name:         "Invalid protocol",
			msg:          []byte("ABCD SET NOT_OK\r\n"),
			expectedResp: response{},
			expectedErr:  ErrUnknownProtocol,
		},
		{
			name: "Valid successful SET response",
			msg:  []byte("RCSP/1.0 SET OK\r\nKEY: key1\r\n"),
			expectedResp: response{
				command: []byte("SET"),
				ok:      true,
				message: nil,
				key:     []byte("key1"),
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Valid unsuccessful SET response",
			msg:  []byte("RCSP/1.0 SET NOT_OK\r\nMESSAGE: Failed to set the key\r\nKEY: key1\r\n"),
			expectedResp: response{
				command: []byte("SET"),
				ok:      false,
				message: []byte("Failed to set the key"),
				key:     []byte("key1"),
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Valid successful GET response",
			msg:  []byte("RCSP/1.0 GET OK\r\nKEY: key1\r\nVALUE: val\r\n"),
			expectedResp: response{
				command: []byte("GET"),
				ok:      true,
				message: nil,
				key:     []byte("key1"),
				value:   []byte("val"),
			},
			expectedErr: nil,
		},
		{
			name: "Valid unsuccessful GET response",
			msg:  []byte("RCSP/1.0 GET NOT_OK\r\nMESSAGE: Not found\r\nKEY: key1\r\n"),
			expectedResp: response{
				command: []byte("GET"),
				ok:      false,
				message: []byte("Not found"),
				key:     []byte("key1"),
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Valid PING response",
			msg:  []byte("RCSP/1.0 PING OK\r\n"),
			expectedResp: response{
				command: []byte("PING"),
				ok:      true,
				message: nil,
				key:     nil,
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Valid successful CLOSE response",
			msg:  []byte("RCSP/1.0 CLOSE OK\r\n"),
			expectedResp: response{
				command: []byte("CLOSE"),
				ok:      true,
				message: nil,
				key:     nil,
				value:   nil,
			},
			expectedErr: nil,
		},
		{
			name: "Generic error response",
			msg:  []byte("RCSP/1.0 NOT_OK\r\nMESSAGE: Unexpected error\r\n"),
			expectedResp: response{
				command: nil,
				ok:      false,
				message: []byte("Unexpected error"),
				key:     nil,
				value:   nil,
			},
			expectedErr: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := parseResponse(tc.msg)
			if err != tc.expectedErr {
				t.Errorf("Expected error value \"%v\", got \"%v\" instead",
					tc.expectedErr, err)
			}
			if bytes.Compare(resp.command, tc.expectedResp.command) != 0 {
				t.Errorf("Expected command \"%s\", got \"%s\" instead",
					string(tc.expectedResp.command), string(resp.command))
			}
			if resp.ok != tc.expectedResp.ok {
				t.Errorf("Expected ok to be \"%v\", got \"%v\" instead",
					tc.expectedResp.ok, resp.ok)
			}
			if bytes.Compare(resp.message, tc.expectedResp.message) != 0 {
				t.Errorf("Expected message \"%s\", got \"%s\" instead",
					string(tc.expectedResp.message), string(resp.message))
			}
			if bytes.Compare(resp.key, tc.expectedResp.key) != 0 {
				t.Errorf("Expected key \"%s\", got \"%s\" instead",
					string(tc.expectedResp.key), string(resp.key))
			}
			if bytes.Compare(resp.value, tc.expectedResp.value) != 0 {
				t.Errorf("Expected value \"%s\", got \"%s\" instead",
					string(tc.expectedResp.value), string(resp.value))
			}
		})
	}
}
