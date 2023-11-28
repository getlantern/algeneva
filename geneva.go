package algeneva

import (
	"context"
	"net"
)

// client is a wrapper around net.Dialer that applies geneva strategies when writing to the connection.
type client struct {
	strategy strategy
}

// NewClient will parse the list of strategies and return a new client. An error is returned if any of the strategies
// are invalid.
func NewClient(strategy string) (*client, error) {
	strat, err := newStrategy(strategy)
	if err != nil {
		return nil, err
	}

	return &client{strategy: strat}, nil
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

	return &Conn{conn: cc, strategy: c.strategy}, nil
}
