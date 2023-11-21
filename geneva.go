package algeneva

import (
	"context"
	"net"
)

// client is a wrapper around net.Dialer that applies geneva strategies when writing to the connection.
type client struct {
	strategies []strategy
}

// NewClient will parse the list of strategies and return a new client. An error is returned if any of the strategies
// are invalid.
func NewClient(strategies []string) (*client, error) {
	ss := make([]strategy, 0, len(strategies))
	for _, s := range strategies {
		strat, err := newStrategy(s)
		if err != nil {
			return nil, err
		}

		ss = append(ss, strat)
	}

	return &client{strategies: ss}, nil
}

// Dial connects to the address on the named network and returns a conn that wraps a net.Conn.
// See net.Dial (https://pkg.go.dev/net#Dial) for more information about network and address parameters.
// Dial uses context.Background() internally; use DialContext to specify a context.
func (c *client) Dial(network, address string) (net.Conn, error) {
	return c.DialContext(context.Background(), network, address)
}

// DialContext connects to the address on the named network using the provided context and returns a conn that wraps a
// net.Conn.
//
// See net.Dial (https://pkg.go.dev/net#Dial) for more information about network and address parameters.
//
// The provided Context must be non-nil. If the context expires before the connection is complete, an error is
// returned. Once successfully connected, any expiration of the context will not affect the connection.
//
// When using TCP, and the host in the address parameter resolves to multiple network addresses, any dial timeout
// (from d.Timeout or ctx) is spread over each consecutive dial, such that each is given an appropriate fraction of
// the time to connect. For example, if a host has 4 IP addresses and the timeout is 1 minute, the connect to each
// single address will be given 15 seconds to complete before trying the next one.
func (c *client) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	cc, err := (&net.Dialer{}).DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return &conn{Conn: cc, strategies: c.strategies}, nil
}
