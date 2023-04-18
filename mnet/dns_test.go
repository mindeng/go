package mnet

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/http/httptrace"
	"testing"
	"time"
)

// go test -v -run TestMultiIPLookup

// TestMultiIPLookup DNS resolution of a name that has multiple A records.
func TestMultiIPLookup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "www.google.com"},
		{name: "www.facebook.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ips, err := net.LookupIP(tt.name)
			if err != nil {
				t.Fatalf("LookupIP(%q) = %v", tt.name, err)
			}
			if len(ips) == 0 {
				t.Fatalf("LookupIP(%q) = no IPs", tt.name)
			}
			log.Printf("LookupIP(%q) = %v", tt.name, ips)
		})
	}
}

// TestHttpReqWithSpecIP HTTP request to a specific IP address for a domain.
func TestHttpReqWithSpecIP(t *testing.T) {
	t.Parallel()

	// This name has multiple A records.
	const name = "www.google.com"

	// Get the IP addresses for the name.
	// Only get ipv4 addresses.
	ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip4", name)
	if err != nil {
		t.Fatalf("LookupIP(%q) = %v", name, err)
	}

	// Create a HTTP request to the name with each IP address.
	for _, ip := range ips {
		// this assignment is needed to avoid the loop variable being reused.
		// see: https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		ip := ip
		t.Run(ip.String(), func(t *testing.T) {
			t.Parallel()
			curlResolveSpecIP(t, "https://"+name, ip.String())
		})
	}
}

// curlResolveSpecIP do a HTTP GET request to the uri with a specific IP address.
// just like: curl --resolve www.google.com:443:<ip> https://www.google.com
func curlResolveSpecIP(t *testing.T, uri, ip string) {
	// Create a custom dialer that will be used in custom transport.
	dialer := &net.Dialer{}

	// Setup a transport with a dialer that uses the first IP address.
	transport := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return dialer.DialContext(context.Background(), network, net.JoinHostPort(ip, "443"))
		},
	}

	// Create a client with the custom transport.
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	// Use httptrace to get the remote IP address of the connection.
	remoteIP := ""
	trace := &httptrace.ClientTrace{
		GotConn: func(connInfo httptrace.GotConnInfo) {
			remoteIP = connInfo.Conn.RemoteAddr().(*net.TCPAddr).IP.String()
			log.Printf("Got connection with IP %q", remoteIP)
		},
	}

	// Create a request with the trace.
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		t.Fatalf("Failed to create HTTP request: %v", err)
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// Do the request.
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP GET request failed: %v", err)
	}
	defer resp.Body.Close()
	log.Printf("HTTP GET request to %q with IP %q succeeded: %s", uri, ip, resp.Status)

	// Check if the remote IP address is the same as the one we used.
	if remoteIP != ip {
		t.Fatalf("Remote IP address %q is not the same as the one we used %q", remoteIP, ip)
	}
}
