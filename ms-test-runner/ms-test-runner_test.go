package main

import "testing"

func TestEventTestsPattern(t *testing.T) {
	events := []*TestEvent{
		{Test: "TestA/1"},
		{Test: "TestA/2"},
		{Test: "TestB/1"},
		{Test: "TestB/2"},
	}
	const expected = `^(TestA|TestB)$`
	s := eventTestsPattern(events)
	if s != expected {
		t.Errorf("Got: `%s` Want: `%s`", s, expected)
	}
}
