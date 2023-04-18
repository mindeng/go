package mnet

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

// TestFastestIP tests the FastestIP function.
func TestFastestIP(t *testing.T) {
	domain := "www.google.com"
	// Get IP addresses by looking up the domain.
	ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip4", domain)
	if err != nil {
		t.Fatal(err)
	}

	// Print the IP addresses.
	fmt.Printf("IP addresses: %q\n", ips)

	// Map ips to a slice of strings.
	ipsStr := make([]string, len(ips))
	for i, ip := range ips {
		ipsStr[i] = net.JoinHostPort(ip.String(), "443")
	}

	// Get the fastest IP address.
	fastestIP, err := FastestAddress(ipsStr)
	if err != nil {
		t.Fatal(err)
	}

	// Print the fastest IP address.
	fmt.Printf("---> fastest IP: %s\n", *fastestIP)

	// Wait for 2 seconds.
	time.Sleep(2 * time.Second)
}
