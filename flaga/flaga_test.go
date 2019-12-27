package flaga

import (
	"testing"

	"src/github.com/go-test/deep"
)

func TestStrings(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expected    Strings
		want        string
		doesContain string
		notContains string
	}{
		{
			name:  "success-one-value",
			value: "a",
			expected: Strings{
				Values:   []string{"a"},
				Assigned: true,
			},
			want:        `[]string{"a"}`,
			doesContain: "a",
			notContains: "b",
		},
		{
			name:  "success-three-values",
			value: "a,b,c",
			expected: Strings{
				Values:   []string{"a", "b", "c"},
				Assigned: true,
			},
			want:        `[]string{"a", "b", "c"}`,
			doesContain: "c",
			notContains: "d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := Strings{}
			sa.Set(tt.value)
			if diff := deep.Equal(tt.expected, sa); diff != nil {
				t.Errorf("String.Set() unexpected: %v", diff)
			}
			if got := sa.String(); got != tt.want {
				t.Errorf("Strings.String() = %q, want %q", got, tt.want)
			}
			if !sa.Contains(tt.doesContain) {
				t.Errorf("Strings.Contains() did not contain %q, %v", tt.doesContain, sa)
			}
			if sa.Contains(tt.notContains) {
				t.Errorf("Strings.Contains() did contain %q impossible value, %v", tt.notContains, sa)
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name        string
		value       string
		expected    String
		want        string
		doesContain string
		notContains string
	}{
		{
			name:  "success-one-value",
			value: "a",
			expected: String{
				Value:    "a",
				Assigned: true,
			},
			want: "a",
		},
		{
			name:  "success-assigned-empty-value",
			value: "",
			expected: String{
				Value:    "",
				Assigned: true,
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := String{}
			sa.Set(tt.value)
			if diff := deep.Equal(tt.expected, sa); diff != nil {
				t.Errorf("String.Set() unexpected: %v", diff)
			}
			if got := sa.String(); got != tt.want {
				t.Errorf("Strings.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
