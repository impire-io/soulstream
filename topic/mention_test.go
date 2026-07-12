package topic

import (
	"reflect"
	"testing"
)

func TestParseMentions(t *testing.T) {
	cases := []struct {
		body string
		want []string
	}{
		{"hi @daan", []string{"daan"}},
		{"@bookkeeper-agent please check", []string{"bookkeeper-agent"}},
		{"trailing punct @bookkeeper-agent!", []string{"bookkeeper-agent"}},
		{"@Daan @@ @ x", nil},                       // uppercase, bare @@, and @ space → nothing
		{"@daan and @daan again", []string{"daan"}}, // de-duplicated
		{"@a talks to @b-c", []string{"a", "b-c"}},
		{"nobody here", nil},
		{"self @daan mentions", []string{"daan"}}, // self-mention is still parsed
	}
	for _, tc := range cases {
		got := ParseMentions(tc.body)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ParseMentions(%q) = %v, want %v", tc.body, got, tc.want)
		}
	}
}
