package filter

import (
	"testing"
)

func TestStringFilter_Validate(t *testing.T) {
	tests := []struct {
		name    string
		filter  *StringFilter
		wantErr bool
	}{
		{
			name:    "nil filter",
			filter:  nil,
			wantErr: false,
		},
		{
			name:    "empty filter",
			filter:  &StringFilter{},
			wantErr: false,
		},
		{
			name: "valid regex",
			filter: &StringFilter{
				Re: "^test.*",
			},
			wantErr: false,
		},
		{
			name: "valid complex regex",
			filter: &StringFilter{
				Re: "\\d+",
			},
			wantErr: false,
		},
		{
			name: "invalid regex",
			filter: &StringFilter{
				Re: "[",
			},
			wantErr: true,
		},
		{
			name: "invalid regex with escape",
			filter: &StringFilter{
				Re: "\\",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("StringFilter.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStringFilter_CheckMatch(t *testing.T) {
	tests := []struct {
		name   string
		filter *StringFilter
		value  string
		want   bool
	}{
		{
			name:   "nil filter",
			filter: nil,
			value:  "test",
			want:   true,
		},
		{
			name:   "empty filter",
			filter: &StringFilter{},
			value:  "test",
			want:   true,
		},
		{
			name: "empty=true matches empty string",
			filter: &StringFilter{
				Empty: true,
			},
			value: "",
			want:  true,
		},
		{
			name: "empty=true does not match non-empty string",
			filter: &StringFilter{
				Empty: true,
			},
			value: "test",
			want:  false,
		},
		{
			name: "notEmpty=true matches non-empty string",
			filter: &StringFilter{
				NotEmpty: true,
			},
			value: "test",
			want:  true,
		},
		{
			name: "notEmpty=true does not match empty string",
			filter: &StringFilter{
				NotEmpty: true,
			},
			value: "",
			want:  false,
		},
		{
			name: "value matches exact value",
			filter: &StringFilter{
				Value: "test",
			},
			value: "test",
			want:  true,
		},
		{
			name: "value does not match different value",
			filter: &StringFilter{
				Value: "test",
			},
			value: "other",
			want:  false,
		},
		{
			name: "values matches when value in array",
			filter: &StringFilter{
				Values: []string{"test1", "test2"},
			},
			value: "test1",
			want:  true,
		},
		{
			name: "values does not match when value not in array",
			filter: &StringFilter{
				Values: []string{"test1", "test2"},
			},
			value: "test3",
			want:  false,
		},
		{
			name: "empty values array matches any value",
			filter: &StringFilter{
				Values: []string{},
			},
			value: "test",
			want:  true,
		},
		{
			name: "regex matches",
			filter: &StringFilter{
				Re: "^test",
			},
			value: "test123",
			want:  true,
		},
		{
			name: "regex does not match",
			filter: &StringFilter{
				Re: "^test",
			},
			value: "abc123",
			want:  false,
		},
		{
			name: "invalid regex returns false",
			filter: &StringFilter{
				Re: "[",
			},
			value: "test",
			want:  false,
		},
		{
			name: "hasPrefix matches",
			filter: &StringFilter{
				HasPrefix: "test",
			},
			value: "test123",
			want:  true,
		},
		{
			name: "hasPrefix does not match",
			filter: &StringFilter{
				HasPrefix: "test",
			},
			value: "abc123",
			want:  false,
		},
		{
			name: "hasSuffix matches",
			filter: &StringFilter{
				HasSuffix: "123",
			},
			value: "test123",
			want:  true,
		},
		{
			name: "hasSuffix does not match",
			filter: &StringFilter{
				HasSuffix: "123",
			},
			value: "test456",
			want:  false,
		},
		{
			name: "contains matches",
			filter: &StringFilter{
				Contains: "est",
			},
			value: "test123",
			want:  true,
		},
		{
			name: "contains does not match",
			filter: &StringFilter{
				Contains: "xyz",
			},
			value: "test123",
			want:  false,
		},
		{
			name: "combined filters all match",
			filter: &StringFilter{
				NotEmpty:  true,
				HasPrefix: "test",
				HasSuffix: "123",
			},
			value: "test123",
			want:  true,
		},
		{
			name: "combined filters one fails",
			filter: &StringFilter{
				NotEmpty:  true,
				HasPrefix: "test",
				HasSuffix: "456",
			},
			value: "test123",
			want:  false,
		},
		{
			name: "complex combination",
			filter: &StringFilter{
				Re:        "^test",
				Contains:  "12",
				HasSuffix: "3",
			},
			value: "test123",
			want:  true,
		},
		{
			name: "regex with digits",
			filter: &StringFilter{
				Re: "\\d+",
			},
			value: "abc123",
			want:  true,
		},
		{
			name: "regex with digits no match",
			filter: &StringFilter{
				Re: "\\d+",
			},
			value: "abcdef",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.CheckMatch(tt.value)
			if got != tt.want {
				t.Fatalf("StringFilter.CheckMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
