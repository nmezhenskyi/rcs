package nativesrv

import (
	"bytes"
	"net"
)

const (
	ErrMalformedRequest  = messageError("malformed request")
	ErrMalformedResponse = messageError("malformed response")
	ErrUnknownProtocol   = messageError("unknown protocol")
	ErrInvalidKey        = messageError("invalid key")
	ErrInvalidValue      = messageError("invalid value")
)

type request struct {
	command []byte
	key     []byte
	value   []byte
}

func (r *request) write(conn net.Conn) (n int, err error) {
	msg := []byte("RCSP/1.0")
	if r.command != nil {
		msg = append(msg, ' ')
		msg = append(msg, r.command...)
		msg = append(msg, []byte("\r\n")...)
	}
	if r.key != nil {
		msg = append(msg, []byte("KEY: ")...)
		msg = append(msg, r.key...)
		msg = append(msg, []byte("\r\n")...)
	}
	if r.value != nil {
		msg = append(msg, []byte("VALUE: ")...)
		msg = append(msg, r.value...)
		msg = append(msg, []byte("\r\n")...)
	}
	return conn.Write(msg)
}

func parseRequest(msg []byte) (request, error) {
	if len(msg) == 0 {
		return request{}, ErrMalformedRequest
	}

	msgLines := bytes.SplitN(msg, []byte("\r\n"), 3)
	linesCount := len(msgLines)
	if linesCount != 0 && len(msgLines[linesCount-1]) == 0 {
		msgLines = msgLines[:linesCount-1]
		linesCount -= 1
	}
	msgLines[linesCount-1] = bytes.TrimSuffix(msgLines[linesCount-1], []byte("\r\n"))
	headerTokens := bytes.Split(msgLines[0], []byte(" "))
	if len(headerTokens) != 2 || !bytes.Equal(headerTokens[0], []byte("RCSP/1.0")) {
		return request{}, ErrUnknownProtocol
	}

	var (
		parsedReq      request
		encounteredErr error
	)

	// Parse Command:
	parsedReq.command = headerTokens[1]
	// Parse Key:
	if linesCount > 1 {
		keyTokens := bytes.SplitN(msgLines[1], []byte(": "), 2)
		if len(keyTokens) != 2 {
			encounteredErr = ErrInvalidKey
		} else if !bytes.Equal(keyTokens[0], []byte("KEY")) {
			encounteredErr = ErrMalformedRequest
		} else {
			parsedReq.key = keyTokens[1]
		}
	}
	// Parse Value:
	if linesCount > 2 {
		valueTokens := bytes.SplitN(msgLines[2], []byte(": "), 2)
		if len(valueTokens) != 2 || !bytes.Equal(valueTokens[0], []byte("VALUE")) {
			encounteredErr = ErrMalformedRequest
		} else {
			parsedReq.value = valueTokens[1]
		}
	}

	return parsedReq, encounteredErr
}

type response struct {
	command []byte
	ok      bool
	message []byte
	key     []byte
	value   []byte
}

func (r *response) write(conn net.Conn) (n int, err error) {
	msg := []byte("RCSP/1.0")
	if r.command != nil {
		msg = append(msg, ' ')
		msg = append(msg, r.command...)
	}
	if r.ok {
		msg = append(msg, []byte(" OK\r\n")...)
	} else {
		msg = append(msg, []byte(" NOT_OK\r\n")...)
	}
	if r.message != nil {
		msg = append(msg, []byte("MESSAGE: ")...)
		msg = append(msg, r.message...)
		msg = append(msg, []byte("\r\n")...)
	}
	if r.key != nil {
		msg = append(msg, []byte("KEY: ")...)
		msg = append(msg, r.key...)
		msg = append(msg, []byte("\r\n")...)
	}
	if r.value != nil {
		msg = append(msg, []byte("VALUE: ")...)
		msg = append(msg, r.value...)
		msg = append(msg, []byte("\r\n")...)
	}
	return conn.Write(msg)
}

func (r *response) writeError(conn net.Conn, command, message []byte) (n int, err error) {
	r.command = command
	r.ok = false
	r.message = message
	r.key = nil
	r.value = nil
	return r.write(conn)
}

func (r *response) writeErrorWithKey(conn net.Conn, command, message, key []byte) (n int, err error) {
	r.command = command
	r.ok = false
	r.message = message
	r.key = key
	r.value = nil
	return r.write(conn)
}

func parseResponse(msg []byte) (response, error) {
	if len(msg) == 0 {
		return response{}, ErrMalformedResponse
	}

	msgLines := bytes.SplitN(msg, []byte("\r\n"), 4)
	linesCount := len(msgLines)
	if linesCount != 0 && len(msgLines[linesCount-1]) == 0 {
		msgLines = msgLines[:linesCount-1]
		linesCount -= 1
	}
	msgLines[linesCount-1] = bytes.TrimSuffix(msgLines[linesCount-1], []byte("\r\n"))
	headerTokens := bytes.Split(msgLines[0], []byte(" "))
	if len(headerTokens) < 2 {
		return response{}, ErrMalformedResponse
	}
	if !bytes.Equal(headerTokens[0], []byte("RCSP/1.0")) {
		return response{}, ErrUnknownProtocol
	}

	var (
		parsedResp     response
		encounteredErr error
	)

	okTokenIndex := 2
	if len(headerTokens) == 2 {
		// Command is missing, so OK will be the second token in the HEADER line
		okTokenIndex = 1
	} else if len(headerTokens) == 3 {
		parsedResp.command = headerTokens[1]
	} else {
		return response{}, ErrMalformedResponse
	}
	if bytes.Equal(headerTokens[okTokenIndex], []byte("OK")) {
		parsedResp.ok = true
	} else if !bytes.Equal(headerTokens[okTokenIndex], []byte("NOT_OK")) {
		encounteredErr = ErrMalformedResponse
	}

ParsingLoop:
	for i := 1; i < len(msgLines); i++ {
		tokenName, tokenValue, found := bytes.Cut(msgLines[i], []byte(": "))
		if !found || len(tokenName) == 0 || len(tokenValue) == 0 {
			encounteredErr = ErrMalformedResponse
			break ParsingLoop
		}
		switch string(tokenName) {
		case "MESSAGE":
			parsedResp.message = tokenValue
		case "KEY":
			parsedResp.key = tokenValue
		case "VALUE":
			parsedResp.value = tokenValue
		default:
			encounteredErr = ErrMalformedResponse
			break ParsingLoop
		}
	}

	return parsedResp, encounteredErr
}

type messageError string

func (err messageError) Error() string { return string(err) }
