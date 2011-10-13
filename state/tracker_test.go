package state

import (
	"testing"
)

func TestNewNick(t *testing.T) {
	st := NewTracker()

	if len(st.nicks) != 0 {
		t.Errorf("Nick list of new tracker is non-zero length.")
	}

	nick := st.NewNick("test1")

	if nick == nil || nick.Nick != "test1" || nick.st != st {
		t.Errorf("Nick object created incorrectly by NewNick.")
	}
	if n, ok := st.nicks["test1"]; !ok || n != nick || len(st.nicks) != 1 {
		t.Errorf("Nick object stored incorrectly by NewNick.")
	}

	if fail := st.NewNick("test1"); fail != nil {
		t.Errorf("Creating duplicate nick did not produce nil return.")
	}
}

func TestGetNick(t *testing.T) {
	st := NewTracker()

	test1 := &Nick{Nick: "test1", st: st}
	st.nicks["test1"] = test1

	if n := st.GetNick("test1"); n != test1 {
		t.Errorf("Incorrect nick returned by GetNick.")
	}
	if n := st.GetNick("test2"); n != nil {
		t.Errorf("Nick unexpectedly returned by GetNick.")
	}
	if len(st.nicks) != 1 {
		t.Errorf("Nick list changed size during GetNick.")
	}
}

func TestReNick(t *testing.T) {
	st := NewTracker()

	test1 := &Nick{Nick: "test1", st: st}
	st.nicks["test1"] = test1

	st.ReNick("test1", "test2")

	if _, ok := st.nicks["test1"]; ok {
		t.Errorf("Nick test1 still exists after ReNick.")
	}
	if n, ok := st.nicks["test2"]; !ok || n != test1 {
		t.Errorf("Nick test2 doesn't exist after ReNick.")
	}
	if len(st.nicks) != 1 {
		t.Errorf("Nick list changed size during ReNick.")
	}

	test2 := &Nick{Nick: "test2", st: st}
	st.nicks["test1"] = test2

	st.ReNick("test1", "test2")
	if n, ok := st.nicks["test2"]; !ok || n != test1 {
		t.Errorf("Nick test2 overwritten/deleted by ReNick.")
	}
	if n, ok := st.nicks["test1"]; !ok || n != test2 {
		t.Errorf("Nick test1 overwritten/deleted by ReNick.")
	}
	if len(st.nicks) != 2 {
		t.Errorf("Nick list changed size during ReNick.")
	}

}

func TestDelNick(t *testing.T) {
	st := NewTracker()

	test1 := &Nick{Nick: "test1", st: st}
	st.nicks["test1"] = test1

	st.DelNick("test1")

	if _, ok := st.nicks["test1"]; ok {
		t.Errorf("Nick test1 still exists after DelNick.")
	}
	if len(st.nicks) != 0 {
		t.Errorf("Nick list still contains nicks after DelNick.")
	}

	st.nicks["test1"] = test1

	st.DelNick("test2")

	if len(st.nicks) != 1 {
		t.Errorf("DelNick had unexpected side-effects.")
	}
}

func TestNewChannel(t *testing.T) {
	st := NewTracker()

	if len(st.chans) != 0 {
		t.Errorf("Channel list of new tracker is non-zero length.")
	}

	test1 := st.NewChannel("#test1")

	if test1 == nil || test1.Name != "#test1" || test1.st != st {
		t.Errorf("Channel object created incorrectly by NewChannel.")
	}
	if c, ok := st.chans["#test1"]; !ok || c != test1 || len(st.chans) != 1 {
		t.Errorf("Channel object stored incorrectly by NewChannel.")
	}

	if fail := st.NewChannel("#test1"); fail != nil {
		t.Errorf("Creating duplicate chan did not produce nil return.")
	}
}

func TestGetChannel(t *testing.T) {
	st := NewTracker()

	test1 := &Channel{Name: "#test1", st: st}
	st.chans["#test1"] = test1

	if c := st.GetChannel("#test1"); c != test1 {
		t.Errorf("Incorrect Channel returned by GetChannel.")
	}
	if c := st.GetChannel("#test2"); c != nil {
		t.Errorf("Channel unexpectedly returned by GetChannel.")
	}
	if len(st.chans) != 1 {
		t.Errorf("Channel list changed size during GetChannel.")
	}
}

func TestDelChannel(t *testing.T) {
	st := NewTracker()

	test1 := &Channel{Name: "#test1", st: st}
	st.chans["#test1"] = test1

	st.DelChannel("#test1")

	if _, ok := st.chans["#test1"]; ok {
		t.Errorf("Channel test1 still exists after DelChannel.")
	}
	if len(st.chans) != 0 {
		t.Errorf("Channel list still contains chans after DelChannel.")
	}

	st.chans["#test1"] = test1

	st.DelChannel("test2")

	if len(st.chans) != 1 {
		t.Errorf("DelChannel had unexpected side-effects.")
	}
}

func TestIsOn(t *testing.T) {
	st := NewTracker()

	nick1 := st.NewNick("test1")
	chan1 := st.NewChannel("#test1")

	if st.IsOn("#test1", "test1") {
		t.Errorf("test1 is not on #test1 (yet)")
	}
	chan1.AddNick(nick1)
	if !st.IsOn("#test1", "test1") {
		t.Errorf("test1 is on #test1 (now)")
	}
}
