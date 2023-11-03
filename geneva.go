package algeneva

import (
	"context"
	"net"
)

type client struct {
	strategies []strategy
}

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

func (c *client) Dial(network, address string) (net.Conn, error) {
	return c.DialContext(context.Background(), network, address)
}

func (c *client) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	cc, err := (&net.Dialer{}).DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return &conn{Conn: cc, strategies: c.strategies}, nil
}

type listener struct {
	net.Listener
}

func Listen(network, address string) (net.Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	return &listener{Listener: l}, nil
}

// Accept waits for and returns the next connection to the listener.
func (l *listener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return &conn{Conn: c}, nil
}

// Close closes the listener.
// Any blocked Accept operations will be unblocked and return errors.
func (l *listener) Close() error {
	return l.Listener.Close()
}

// Addr returns the listener's network address.
func (l *listener) Addr() net.Addr {
	return l.Listener.Addr()
}
