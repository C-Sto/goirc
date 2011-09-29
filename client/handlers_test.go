package client

import (
	"testing"
)

// This test performs a simple end-to-end verification of correct line parsing
// and event dispatch as well as testing the PING handler. All the other tests
// in this file will call their respective handlers synchronously, otherwise
// testing becomes more difficult.
func TestPING(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)
	m.Send("PING :1234567890")
	m.Expect("PONG :1234567890")
}

// Test the handler for 001 / RPL_WELCOME
func Test001(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Setup a mock event dispatcher to test correct triggering of "connected"
	flag := false
	c.Dispatcher = WasEventDispatched("connected", &flag)

	// Call handler with a valid 001 line
	c.h_001(parseLine(":irc.server.org 001 test :Welcome to IRC test!ident@somehost.com"))
	// Should result in no response to server
	m.ExpectNothing()

	// Check that the event was dispatched correctly
	if !flag {
		t.Errorf("Sending 001 didn't result in dispatch of connected event.")
	}

	// Check host parsed correctly
	if c.Me.Host != "somehost.com" {
		t.Errorf("Host parsing failed, host is '%s'.", c.Me.Host)
	}
}

// Test the handler for 433 / ERR_NICKNAMEINUSE
func Test433(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Call handler with a 433 line, not triggering c.Me.Renick()
	c.h_433(parseLine(":irc.server.org 433 test new :Nickname is already in use."))
	m.Expect("NICK new_")

	// In this case, we're expecting the server to send a NICK line
	if c.Me.Nick != "test" {
		t.Errorf("ReNick() called unexpectedly, Nick == '%s'.", c.Me.Nick)
	}

	// Send a line that will trigger a renick. This happens when our wanted
	// nick is unavailable during initial negotiation, so we must choose a
	// different one before the connection can proceed. No NICK line will be
	// sent by the server to confirm nick change in this case.
	c.h_433(parseLine(":irc.server.org 433 test test :Nickname is already in use."))
	m.Expect("NICK test_")

	if c.Me.Nick != "test_" {
		t.Errorf("ReNick() not called, Nick == '%s'.", c.Me.Nick)
	}
}

// Test the handler for NICK messages
func TestNICK(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Call handler with a NICK line changing "our" nick to test1.
	c.h_NICK(parseLine(":test!test@somehost.com NICK :test1"))
	// Should generate no response to server
	m.ExpectNothing()

	// Verify that our Nick has changed
	if c.Me.Nick != "test1" {
		t.Errorf("NICK did not result in changing our nick.")
	}

	// Create a "known" nick other than ours
	user1 := c.NewNick("user1", "ident1", "name one", "host1.com")

	// Call handler with a NICK line changing user1 to somebody
	c.h_NICK(parseLine(":user1!ident1@host1.com NICK :somebody"))
	m.ExpectNothing()

	if c.GetNick("user1") != nil {
		t.Errorf("Still have a valid Nick for 'user1'.")
	}
	if n := c.GetNick("somebody"); n != user1 {
		t.Errorf("GetNick(somebody) didn't result in correct Nick.")
	}

	// Send a NICK line for an unknown nick.
	c.h_NICK(parseLine(":blah!moo@cows.com NICK :milk"))
	m.ExpectNothing()
}

// Test the handler for CTCP messages
func TestCTCP(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Call handler with CTCP VERSION
	c.h_CTCP(parseLine(":blah!moo@cows.com PRIVMSG test :\001VERSION\001"))

	// Expect a version reply
	m.Expect("NOTICE blah :\001VERSION powered by goirc...\001")

	// Call handler with CTCP PING
	c.h_CTCP(parseLine(":blah!moo@cows.com PRIVMSG test :\001PING 1234567890\001"))

	// Expect a ping reply
	m.Expect("NOTICE blah :\001PING 1234567890\001")

	// Call handler with CTCP UNKNOWN
	c.h_CTCP(parseLine(":blah!moo@cows.com PRIVMSG test :\001UNKNOWN ctcp\001"))

	// Expect nothing in reply
	m.ExpectNothing()
}

// Test the handler for JOIN messages
func TestJOIN(t *testing.T) {
	// TODO(fluffle): Without mocking to ensure that the various methods
	// h_JOIN uses are called, we must check they do the right thing by
	// verifying their expected side-effects instead. Fixing this requires
	// significant effort to move Conn to being a mockable interface type
	// instead of a concrete struct. I'm not sure how feasible this is :-/
	// 
	// Soon, we'll find out :-)

	m, c := setUp(t)
	defer tearDown(m, c)

	// Use #test1 to test expected behaviour
	// Call handler with JOIN by test to #test1
	c.h_JOIN(parseLine(":test!test@somehost.com JOIN :#test1"))

	// Verify that the MODE and WHO commands are sent correctly
	m.Expect("MODE #test1")
	m.Expect("WHO #test1")

	// Simple verification that NewChannel was called for #test1
	test1 := c.GetChannel("#test1")
	if test1 == nil {
		t.Errorf("No Channel for #test1 created on JOIN.")
	}

	// OK, now #test1 exists, JOIN another user we don't know about
	c.h_JOIN(parseLine(":user1!ident1@host1.com JOIN :#test1"))

	// Verify that the WHO command is sent correctly
	m.Expect("WHO user1")

	// Simple verification that NewNick was called for user1
	user1 := c.GetNick("user1")
	if user1 == nil {
		t.Errorf("No Nick for user1 created on JOIN.")
	}

	// Now, JOIN a nick we *do* know about.
	user2 := c.NewNick("user2", "ident2", "name two", "host2.com")
	c.h_JOIN(parseLine(":user2!ident2@host2.com JOIN :#test1"))

	// We already know about this user and channel, so nothing should be sent
	m.ExpectNothing()

	// Simple verification that the state tracking has actually been done
	if _, ok := test1.Nicks[user2]; !ok || len(test1.Nicks) != 3 {
		t.Errorf("State tracking horked, hopefully other unit tests fail.")
	}

	// Test error paths -- unknown channel, unknown nick
	c.h_JOIN(parseLine(":blah!moo@cows.com JOIN :#test2"))
	m.ExpectNothing()

	// unknown channel, known nick that isn't Me.
	c.h_JOIN(parseLine(":user2!ident2@host2.com JOIN :#test2"))
	m.ExpectNothing()
}

// Test the handler for PART messages
func TestPART(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create user1 and add them to #test1 and #test2
	user1 := c.NewNick("user1", "ident1", "name one", "host1.com")
	test1 := c.NewChannel("#test1")
	test2 := c.NewChannel("#test2")
	test1.AddNick(user1)
	test2.AddNick(user1)

	// Add Me to both channels (not strictly necessary)
	test1.AddNick(c.Me)
	test2.AddNick(c.Me)

	// Then make them PART
	c.h_PART(parseLine(":user1!ident1@host1.com PART #test1 :Bye!"))

	// Expect no output
	m.ExpectNothing()

	// Quick check of tracking code
	if len(test1.Nicks) != 1 {
		t.Errorf("PART failed to remove user1 from #test1.")
	}

	// Test error states.
	// Part a known user from a known channel they are not on.
	c.h_PART(parseLine(":user1!ident1@host1.com PART #test1 :Bye!"))

	// Part an unknown user from a known channel.
	c.h_PART(parseLine(":user2!ident2@host2.com PART #test1 :Bye!"))

	// Part a known user from an unknown channel.
	c.h_PART(parseLine(":user1!ident1@host1.com PART #test3 :Bye!"))

	// Part an unknown user from an unknown channel.
	c.h_PART(parseLine(":user2!ident2@host2.com PART #test3 :Bye!"))
}

// Test the handler for KICK messages
// (this is very similar to the PART message test)
func TestKICK(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create user1 and add them to #test1 and #test2
	user1 := c.NewNick("user1", "ident1", "name one", "host1.com")
	test1 := c.NewChannel("#test1")
	test2 := c.NewChannel("#test2")
	test1.AddNick(user1)
	test2.AddNick(user1)

	// Add Me to both channels (not strictly necessary)
	test1.AddNick(c.Me)
	test2.AddNick(c.Me)

	// Then kick them!
	c.h_KICK(parseLine(":test!test@somehost.com KICK #test1 user1 :Bye!"))

	// Expect no output
	m.ExpectNothing()

	// Quick check of tracking code
	if len(test1.Nicks) != 1 {
		t.Errorf("PART failed to remove user1 from #test1.")
	}

	// Test error states.
	// Kick a known user from a known channel they are not on.
	c.h_KICK(parseLine(":test!test@somehost.com KICK #test1 user1 :Bye!"))

	// Kick an unknown user from a known channel.
	c.h_KICK(parseLine(":test!test@somehost.com KICK #test2 user2 :Bye!"))

	// Kick a known user from an unknown channel.
	c.h_KICK(parseLine(":test!test@somehost.com KICK #test3 user1 :Bye!"))

	// Kick an unknown user from an unknown channel.
	c.h_KICK(parseLine(":test!test@somehost.com KICK #test4 user2 :Bye!"))
}

// Test the handler for QUIT messages
func TestQUIT(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create user1 and add them to #test1 and #test2
	user1 := c.NewNick("user1", "ident1", "name one", "host1.com")
	test1 := c.NewChannel("#test1")
	test2 := c.NewChannel("#test2")
	test1.AddNick(user1)
	test2.AddNick(user1)

	// Add Me to both channels (not strictly necessary)
	test1.AddNick(c.Me)
	test2.AddNick(c.Me)

	// Have user1 QUIT
	c.h_QUIT(parseLine(":user1!ident1@host1.com QUIT :Bye!"))

	// Expect no output
	m.ExpectNothing()

	// Quick check of tracking code
	if len(test1.Nicks) != 1 || len(test2.Nicks) != 1 {
		t.Errorf("QUIT failed to remove user1 from channels.")
	}

	// Ensure user1 is no longer a known nick
	if c.GetNick("user1") != nil {
		t.Errorf("QUIT failed to remove user1 from state tracking completely.")
	}

	// Have user1 QUIT again, expect ERRORS!
	c.h_QUIT(parseLine(":user1!ident1@host1.com QUIT :Bye!"))

	// Have a previously unmentioned user quit, expect an error
	c.h_QUIT(parseLine(":user2!ident2@host2.com QUIT :Bye!"))
}

// Test the handler for MODE messages
func TestMODE(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create user1 and add them to #test1
	user1 := c.NewNick("user1", "ident1", "name one", "host1.com")
	test1 := c.NewChannel("#test1")
	test1.AddNick(user1)
	test1.AddNick(c.Me)
	cm := test1.Modes

	// Verify the ChanPrivs exists and modes we're testing aren't set
	if cp, ok := user1.Channels[test1]; !ok || c.Me.Channels[test1].Voice ||
		cp.Op || cm.Key != "" || cm.InviteOnly || cm.Secret {
		t.Errorf("Channel privileges in unexpected state before MODE.")
	}

	// Send a channel mode line
	c.h_MODE(parseLine(":user1!ident1@host1.com MODE #test1 +kisvo somekey test user1"))

	// Expect no output
	m.ExpectNothing()

	// Verify expected state afterwards.
	if cp := user1.Channels[test1]; !(cp.Op || c.Me.Channels[test1].Voice ||
		cm.Key != "somekey" || cm.InviteOnly || cm.Secret) {
		t.Errorf("Channel privileges in unexpected state after MODE.")
	}

	// Verify our nick modes are what we expect before test
	nm := c.Me.Modes
	if nm.Invisible || nm.WallOps || nm.HiddenHost {
		t.Errorf("Our nick privileges in unexpected state before MODE.")
	}

	// Send a nick mode line
	c.h_MODE(parseLine(":test!test@somehost.com MODE test +ix"))
	m.ExpectNothing()

	// Verify the two modes we expect to change did so
	if !nm.Invisible || nm.WallOps || !nm.HiddenHost {
		t.Errorf("Our nick privileges in unexpected state after MODE.")
	}

	// Check error paths -- send a valid user mode that's not us
	c.h_MODE(parseLine(":user1!ident1@host1.com MODE user1 +w"))
	m.ExpectNothing()

	// Send a random mode for an unknown channel
	c.h_MODE(parseLine(":user1!ident1@host1.com MODE #test2 +is"))
	m.ExpectNothing()
}

// Test the handler for TOPIC messages
func TestTOPIC(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create #test1 so we have a channel to set the topic of
	test1 := c.NewChannel("#test1")

	// Assert that it has no topic originally
	if test1.Topic != "" {
		t.Errorf("Test channel already has a topic.")
	}

	// Send a TOPIC line
	c.h_TOPIC(parseLine(":user1!ident1@host1.com TOPIC #test1 :something something"))
	m.ExpectNothing()

	// Make sure the channel's topic has been changed
	if test1.Topic != "something something" {
		t.Errorf("Topic of test channel not set correctly.")
	}

	// Check error paths -- send a topic for an unknown channel
	c.h_TOPIC(parseLine(":user1!ident1@host1.com TOPIC #test2 :dark side"))
	m.ExpectNothing()
}

// Test the handler for 311 / RPL_WHOISUSER
func Test311(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create user1, who we know little about
	user1 := c.NewNick("user1", "", "", "")

	// Send a 311 reply
	c.h_311(parseLine(":irc.server.org 311 test user1 ident1 host1.com * :name"))
	m.ExpectNothing()

	// Verify we now know more about user1
	if user1.Ident != "ident1" ||
		user1.Host != "host1.com" ||
		user1.Name != "name" {
		t.Errorf("WHOIS info of user1 not set correctly.")
	}

	// Check error paths -- send a 311 for an unknown nick
	c.h_311(parseLine(":irc.server.org 311 test user2 ident2 host2.com * :dongs"))
	m.ExpectNothing()
}

// Test the handler for 324 / RPL_CHANNELMODEIS
func Test324(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create #test1, whose modes we don't know
	test1 := c.NewChannel("#test1")
	cm := test1.Modes

	// Make sure modes are unset first
	if cm.Secret || cm.NoExternalMsg || cm.Moderated || cm.Key != "" {
		t.Errorf("Channel modes unexpectedly set before 324 reply.")
	}

	// Send a 324 reply
	c.h_324(parseLine(":irc.server.org 324 test #test1 +snk somekey"))
	m.ExpectNothing()

	// Make sure the modes we expected to be set were set and vice versa
	if !cm.Secret || !cm.NoExternalMsg || cm.Moderated || cm.Key != "somekey" {
		t.Errorf("Channel modes unexpectedly set before 324 reply.")
	}

	// Check unknown channel causes an error
	c.h_324(parseLine(":irc.server.org 324 test #test2 +pmt"))
	m.ExpectNothing()
}

// Test the handler for 332 / RPL_TOPIC
func Test332(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create #test1, whose topic we don't know
	test1 := c.NewChannel("#test1")

	// Assert that it has no topic originally
	if test1.Topic != "" {
		t.Errorf("Test channel already has a topic.")
	}

	// Send a 332 reply
	c.h_332(parseLine(":irc.server.org 332 test #test1 :something something"))
	m.ExpectNothing()

	// Make sure the channel's topic has been changed
	if test1.Topic != "something something" {
		t.Errorf("Topic of test channel not set correctly.")
	}

	// Check unknown channel causes an error
	c.h_324(parseLine(":irc.server.org 332 test #test2 :dark side"))
	m.ExpectNothing()
}

// Test the handler for 352 / RPL_WHOREPLY
func Test352(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create user1, who we know little about
	user1 := c.NewNick("user1", "", "", "")

	// Send a 352 reply
	c.h_352(parseLine(":irc.server.org 352 test #test1 ident1 host1.com irc.server.org user1 G :0 name"))
	m.ExpectNothing()

	// Verify we now know more about user1
	if user1.Ident != "ident1" ||
		user1.Host != "host1.com" ||
		user1.Name != "name" ||
		user1.Modes.Invisible ||
		user1.Modes.Oper {
		t.Errorf("WHO info of user1 not set correctly.")
	}

	// Check that modes are set correctly from WHOREPLY
	c.h_352(parseLine(":irc.server.org 352 test #test1 ident1 host1.com irc.server.org user1 H* :0 name"))
	m.ExpectNothing()

	if !user1.Modes.Invisible || !user1.Modes.Oper {
		t.Errorf("WHO modes of user1 not set correctly.")
	}

	// Check error paths -- send a 352 for an unknown nick
	c.h_352(parseLine(":irc.server.org 352 test #test2 ident2 host2.com irc.server.org user2 G :0 fooo"))
	m.ExpectNothing()
}

// Test the handler for 353 / RPL_NAMREPLY
func Test353(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create #test1, whose user list we're mostly unfamiliar with
	test1 := c.NewChannel("#test1")
	user1 := c.NewNick("user1", "ident1", "name one", "host1.com")
	test1.AddNick(user1)
	test1.AddNick(c.Me)

	// lazy lazy lazy ;-)
	get := func(n string) *ChanPrivs {
		if p, ok := test1.Nicks[c.GetNick(n)]; ok {
			return p
		}
		return nil
	}

	// Verify the lack of nicks
	if len(test1.Nicks) != 2 {
		t.Errorf("Unexpected number of nicks in test channel before 353.")
	}

	// Verify that user1 isn't opped yet
	if p := get("user1"); p == nil || p.Op {
		t.Errorf("Unexpected permissions for user1 before 353.")
	}

	// Send a couple of names replies (complete with trailing space), expect no errors
	c.h_353(parseLine(":irc.server.org 353 test = #test1 :test @user1 user2 +voice "))
	c.h_353(parseLine(":irc.server.org 353 test = #test1 :%halfop @op &admin ~owner "))
	m.ExpectNothing()

	if len(test1.Nicks) != 8 {
		t.Errorf("Unexpected number of nicks in test channel after 353.")
	}

	// TODO(fluffle): Testing side-effects is starting to get on my tits.
	// As a result, this makes some assumptions about the implementation of
	// h_353 that may or may not be valid in the future. Hopefully, I will have
	// time to rewrite the nick / channel handling soon.
	if p := get("user1"); p == nil || !p.Op {
		t.Errorf("353 handler failed to op known nick user1.")
	}

	if p := get("user2"); p == nil || p.Voice || p.HalfOp || p.Op || p.Admin || p.Owner {
		t.Errorf("353 handler set modes on new nick user2.")
	}

	if p := get("voice"); p == nil || !p.Voice {
		t.Errorf("353 handler failed to parse voice correctly.")
	}

	if p := get("halfop"); p == nil || !p.HalfOp {
		t.Errorf("353 handler failed to parse halfop correctly.")
	}

	if p := get("op"); p == nil || !p.Op {
		t.Errorf("353 handler failed to parse op correctly.")
	}

	if p := get("admin"); p == nil || !p.Admin {
		t.Errorf("353 handler failed to parse admin correctly.")
	}

	if p := get("owner"); p == nil || !p.Owner {
		t.Errorf("353 handler failed to parse owner correctly.")
	}

	// Check unknown channel causes an error
	c.h_324(parseLine(":irc.server.org 353 test = #test2 :test ~user3"))
	m.ExpectNothing()
}

// Test the handler for 671 (unreal specific)
func Test671(t *testing.T) {
	m, c := setUp(t)
	defer tearDown(m, c)

	// Create user1, who should not be secure
	user1 := c.NewNick("user1", "ident1", "name one", "host1.com")
	if user1.Modes.SSL {
		t.Errorf("Test nick user1 is already using SSL?")
	}

	// Send a 671 reply
	c.h_671(parseLine(":irc.server.org 671 test user1 :some ignored text"))
	m.ExpectNothing()

	// Ensure user1 is now known to be on an SSL connection
	if !user1.Modes.SSL {
		t.Errorf("Test nick user1 not using SSL?")
	}

	// Check error paths -- send a 671 for an unknown nick
	c.h_671(parseLine(":irc.server.org 671 test user2 :some ignored text"))
	m.ExpectNothing()
}
