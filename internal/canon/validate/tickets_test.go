package validate

import (
	"reflect"
	"sort"
	"testing"
)

func TestKnownProviders(t *testing.T) {
	got := KnownProviders()
	want := []string{"azure-devops", "github", "jira", "linear", "servicenow"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("KnownProviders() = %v, want %v", got, want)
	}
	if !sort.StringsAreSorted(got) {
		t.Errorf("KnownProviders() not sorted: %v", got)
	}
	// Every returned key must index a real validator entry; "none"/"" excluded.
	for _, p := range got {
		if _, ok := ticketValidators[p]; !ok {
			t.Errorf("KnownProviders() returned %q which is not in ticketValidators", p)
		}
		if p == "none" || p == "" {
			t.Errorf("KnownProviders() must not return %q", p)
		}
	}
	if len(got) != len(ticketValidators) {
		t.Errorf("KnownProviders() returned %d keys, ticketValidators has %d", len(got), len(ticketValidators))
	}
}
