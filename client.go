package main

import (
	"./irc/_obj/irc";
	"fmt";
	"os";
	"bufio";
	"strings";
)

func main() {
	// create new IRC connection
	c := irc.New("GoTest", "gotest", "GoBot");
	c.AddHandler("connected",
		func(conn *irc.Conn, line *irc.Line) {
			conn.Join("#");
		}
	);

	// connect to server
	if err := c.Connect("irc.pl0rt.org", ""); err != nil {
		fmt.Printf("Connection error: %s\n", err);
		return;
	}

	// set up a goroutine to read commands from stdin
	in := make(chan string, 4);
	reallyquit := false;
	go func() {
		con := bufio.NewReader(os.Stdin);
		for {
			s, err := con.ReadString('\n');
			if err != nil {
				// wha?, maybe ctrl-D...
				close(in);
				break;
			}
			// no point in sending empty lines down the channel
			if len(s) > 2 {
				in <- s[0:len(s)-1]
			}
		}
	}();

	// set up a goroutine to do parsey things with the stuff from stdin
	go func() {
		for {
			if closed(in) {
				break;
			}
			cmd := <-in;
			if cmd[0] == ':' {
				switch idx := strings.Index(cmd, " "); {
					case idx == -1:
						continue;
					case cmd[1] == 'q':
						reallyquit = true;
						c.Quit(cmd[idx+1:len(cmd)]);
					case cmd[1] == 'j':
						c.Join(cmd[idx+1:len(cmd)]);
					case cmd[1] == 'p':
						c.Part(cmd[idx+1:len(cmd)]);
					case cmd[1] == 'd':
						fmt.Printf(c.String());
				}
			} else {
				c.Raw(cmd)
			}
		}
	}();

	// stall here waiting for asplode on error channel
	for {
		if closed(c.Err) {
			// c.Err being closed indicates we've been disconnected from the
			// server for some reason (e.g. quit, kill or ping timeout)
			// if we don't really want to quit, reconnect!
			if !reallyquit {
				fmt.Println("Reconnecting...");
				if err := c.Connect("irc.pl0rt.org", ""); err != nil {
					fmt.Printf("Connection error: %s\n", err);
					break;
				}
				continue;
			}
			break;
		}
		if err := <-c.Err; err != nil {
			fmt.Printf("goirc error: %s\n", err);
		}
	}
}
