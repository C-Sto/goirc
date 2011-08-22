package client

import "testing"

func setUp(t *testing.T) (*mockNetConn, *Conn) {
	c := New("test", "test", "Testing IRC")
	m := MockNetConn(t)
	c.sock = m
	c.Flood = true // Tests can take a while otherwise
	c.postConnect()
	return m, c
}

// Mock dispatcher to verify that events are triggered successfully
type mockDispatcher func(string, ...interface{})

func (d mockDispatcher) Dispatch(name string, ev ...interface{}) {
	d(name, ev...)
}
