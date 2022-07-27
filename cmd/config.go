package main

import (
	"encoding/json"
	"os"
)

type nativeConf struct {
	Activate    bool   `json:"activate"`    // If true starts the Native server.
	Port        int    `json:"port"`        // Port to listen on.
	OnLocalhost bool   `json:"onLocalhost"` // If true starts listening on localhost.
	TLS         bool   `json:"tls"`         // Enables TLS connections (requires cert & key files).
	CertFile    string `json:"certFile"`    // Path to the TLS/SSL certificate file.
	KeyFile     string `json:"keyFile"`     // Path to the TLS/SSL key file.
}

type grpcConf struct {
	Activate    bool   `json:"activate"`    // If true starts the GRPC server.
	Port        int    `json:"port"`        // Port to listen on.
	OnLocalhost bool   `json:"onLocalhost"` // If true starts listening on localhost.
	TLS         bool   `json:"tls"`         // Enables TLS connections (requires cert & key files).
	CertFile    string `json:"certFile"`    // Path to the TLS/SSL certificate file.
	KeyFile     string `json:"keyFile"`     // Path to the TLS/SSL key file.
}

type httpConf struct {
	Activate    bool   `json:"activate"`    // If true starts the HTTP server.
	Port        int    `json:"port"`        // Port to listen on.
	OnLocalhost bool   `json:"onLocalhost"` // If true starts listening on localhost.
	TLS         bool   `json:"tls"`         // Enables TLS connections (requires cert & key files).
	CertFile    string `json:"certFile"`    // Path to the TLS/SSL certificate file.
	KeyFile     string `json:"keyFile"`     // Path to the TLS/SSL key file.
}

// config contains configurable settings for the program.
type config struct {
	Native    nativeConf `json:"native"`    // Settings for Native server.
	GRPC      grpcConf   `json:"grpc"`      // Settings for GRPC server.
	HTTP      httpConf   `json:"http"`      // Settings for HTTP server.
	Verbosity string     `json:"verbosity"` // Accepted values: "prod", "dev", or "none".
}

// readConfig reads the configurating file and initializes config struct with its
// contents. Returns error on failed read or if the data in file is corrupted.
func readConfig(filename string) (*config, error) {
	byteData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	conf := &config{}
	err = json.Unmarshal(byteData, conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}
