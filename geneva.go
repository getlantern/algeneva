package algeneva

import (
	"context"
	"net"
)

// Client is a wrapper around net.Dialer that applies geneva strategies when writing to the connection.
type Client struct {
	strategy strategy
}

// NewClient will parse the list of strategies and return a new client. An error is returned if any of the strategies
// are invalid.
func NewClient(strategy string) (*Client, error) {
	strat, err := newStrategy(strategy)
	if err != nil {
		return nil, err
	}

	return &Client{strategy: strat}, nil
}

// Dial connects to the address on the named network and then wraps the resulting connection in a net.Conn that
// applies the configured strategy to http requests sent on the connection.
func (c *Client) Dial(network, address string) (net.Conn, error) {
	return c.DialContext(context.Background(), network, address)
}

// DialContext connects to the address on the named network using the provided context and then wraps the resulting
// connection in a net.Conn that applies the configured strategy to http requests sent on the connection.
//
// The provided Context must be non-nil. If the context expires before the connection is complete, an error is
// returned. Once successfully connected, any expiration of the context will not affect the connection.
func (c *Client) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return c.DialContextWithDialer(ctx, &net.Dialer{}, network, address)
}

// DialWithDialer connects to the address on the named network using dialer.Dial and then wraps the resulting
// connection in a net.Conn that applies the configured strategy to http requests sent on the connection.
func (c *Client) DialWithDialer(dialer *net.Dialer, network, address string) (net.Conn, error) {
	return c.DialContextWithDialer(context.Background(), dialer, network, address)
}

// DialContextWithDialer connects to the address on the named network using dialer.DialContext and the provided
// context and then wraps the resulting connection in a net.Conn that applies the configured strategy to http requests
// sent on the connection.
func (c *Client) DialContextWithDialer(ctx context.Context, dialer *net.Dialer, network, address string) (net.Conn, error) {
	cc, err := dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return &conn{Conn: cc, strategy: c.strategy}, nil
}
