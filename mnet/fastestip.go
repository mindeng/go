package mnet

import (
	"errors"
	"fmt"
	"net"
	"time"
)

type Result struct {
	Name  string
	Delay time.Duration
}

// FastestAddress returns the fastest address from the specified address list.
// If the address list is empty, it returns error.
func FastestAddress(addresses []string) (*Result, error) {
	if len(addresses) == 0 {
		return nil, errors.New("empty address list")
	}
	return first(connDelay, addresses)
}

type delayTestFunc func(string) (time.Duration, error)

// first call DelayTestFunc on every address in the list and return the fastest one.
func first(delayTest delayTestFunc, addresses []string) (*Result, error) {
	c := make(chan *Result)
	doDelayTest := func(address string) {
		delay, err := delayTest(address)
		if err == nil {
			// Print the delay time for each address
			fmt.Printf("%s: %s\n", address, delay)
			c <- &Result{address, delay}
		} else {
			fmt.Printf("%s: %s\n", address, err)
		}
	}

	for _, addr := range addresses {
		go doDelayTest(addr)
	}

	// peek the first result, and return the fastest one, with 5 seconds timeout
	select {
	case result := <-c:
		return result, nil
	case <-time.After(5 * time.Second):
		return nil, errors.New("timeout")
	}

}

// connDelay try to connect to the specified address and return the delay time in time.Duration.
// If the address is not reachable, it returns error.
func connDelay(address string) (time.Duration, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return time.Since(start), nil
}
