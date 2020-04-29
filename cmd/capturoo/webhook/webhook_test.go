package webhook

import (
	"reflect"
	"testing"
)

func TestDisplayEvents(t *testing.T) {
	events := []string{
		"bucket.created",
		"bucket.deleted",
		"lead.created:skincare",
	}
	expected := "['bucket.created', 'bucket.deleted', 'lead.created:skincare']"
	result := displayEvents(events)
	if result != expected {
		t.Errorf("displayEvents(events) incorrect, got: %s, want: %s", result, expected)
	}
}

func TestEnabledDisabled(t *testing.T) {
	result := enabledDisabled(true)
	if result != "Enabled" {
		t.Errorf("enabledDisabled(true) incorrect, got: %s, want: %s.", result, "Enabled")
	}

	result = enabledDisabled(false)
	if result != "Disabled" {
		t.Errorf("enabledDisabled(false) incorrect, got: %s, want: %s.", result, "Disabled")
	}

}

func TestParseEventArgs(t *testing.T) {
	s := "bucket.created,bucket.deleted,lead.created:bucket-one|bucket-two|bucket-three"
	events, err := parseEventArgs(s)
	if err != nil {
		t.Fatalf("parseEventArgs(%q) returned an error", s)
	}
	if len(events) != 3 {
		t.Fatalf("len(events) incorrect, got: %d, want: %d", len(events), 3)
	}

	// 0
	if events[0].name != "bucket.created" {
		t.Errorf("events[0].name incorrect, got: %q, want: %q", events[0].name, "bucket.created")
	}
	if events[0].resources != nil {
		t.Errorf("events[0].resource incorrect, got: %v, want: %v", events[0].resources, nil)
	}

	// 1
	if events[1].name != "bucket.deleted" {
		t.Errorf("events[1].name incorrect, got: %q, want: %q", events[1].name, "bucket.deleted")
	}
	if events[1].resources != nil {
		t.Errorf("events[1].resource incorrect, got: %v, want: %v", events[1].resources, nil)
	}

	// 2
	if events[2].name != "lead.created" {
		t.Errorf("events[2].name incorrect, got: %q, want: %q", events[2].name, "lead.created")
	}
	want := []string{"bucket-one", "bucket-two", "bucket-three"}
	if !reflect.DeepEqual(events[2].resources, want) {
		t.Errorf("events[2].resources incorrect, got: %v, want: %v", events[2].resources, want)
	}
}

// func TestUnknownEvents(t *testing.T) {
// 	events := []string{
// 		"bucket.created",
// 		"bucket.deleted",
// 		"lead.created:bucket-one|bucket-two",
// 	}
// 	unknowns := unknownEvents(events)
// 	if unknowns != nil {
// 		t.Errorf("unknownEvents(%v) incorrect, got: %v, want: %v", events, unknowns, nil)
// 	}

// 	events = []string{
// 		"bucket.created",
// 		"non-event-one",
// 		"bucket.deleted",
// 		"lead.created:non-event-two",
// 		"lead.created:bucket-one|bucket-two",
// 	}
// 	want := []string{
// 		"non-event-one",
// 		"non-event-two",
// 	}
// 	unknowns = unknownEvents(events)
// 	if !reflect.DeepEqual(unknowns, want) {
// 		t.Errorf("unknownEvents(%v) incorrect, got: %v, want: %v", events, unknowns, want)
// 	}

// 	fmt.Printf("%#v\n", unknowns)
// 	fmt.Printf("%#v\n", want)
// }

func TestIsContextDriven(t *testing.T) {
	s := "lead.created:bucket-one|bucket-two|bucket-three"
	result := isContextDriven(s)
	if !result {
		t.Errorf("isContextDriven(%q) incorrect, got: %t, want: %t", s, result, true)
	}

	s = "bucket.created"
	result = isContextDriven(s)
	if result {
		t.Errorf("isContextDriven(%q) incorrect, got: %t, want: %t", s, result, false)
	}
}

func TestContains(t *testing.T) {
	fruits := []string{
		"apples",
		"oranges",
		"bananas",
		"pears",
		"grapes",
	}
	result := contains(fruits, "bananas")
	if !result {
		t.Errorf("contains(%q) incorect, got: %t, want: %t", fruits, result, true)
	}

	result = contains(fruits, "pinapple")
	if result {
		t.Errorf("contains(%q) incorrect, got: %t, want: %t", fruits, result, false)
	}
}
