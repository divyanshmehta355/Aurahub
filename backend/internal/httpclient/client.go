package httpclient

import (
	"net"
	"net/http"
	"time"
)

var (
	// SharedTransport provides optimized connection pooling across the entire application.
	SharedTransport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   25,
		MaxConnsPerHost:       100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
	}

	// Default is the shared client for standard API requests (status checks, metadata lookups).
	Default = &http.Client{
		Transport: SharedTransport,
		Timeout:   25 * time.Second,
	}

	// Upload is the client tuned for large media uploads and chunk submissions.
	Upload = &http.Client{
		Transport: SharedTransport,
		Timeout:   5 * time.Minute,
	}
)
