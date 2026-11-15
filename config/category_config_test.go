package config

import (
	"testing"

	"github.com/BurntSushi/toml"
)

func TestCategoryConfigDescending(t *testing.T) {
	trueValue := true
	falseValue := false
	tests := []struct {
		name  string
		input string
		want  *bool
	}{
		{name: "unset", input: "", want: nil},
		{name: "true", input: "descending = true", want: &trueValue},
		{name: "false", input: "descending = false", want: &falseValue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var conf CategoryConfig
			if _, err := toml.Decode(tt.input, &conf); err != nil {
				t.Fatal(err)
			}
			switch {
			case tt.want == nil && conf.Descending != nil:
				t.Fatalf("Descending = %v, want nil", *conf.Descending)
			case tt.want != nil && conf.Descending == nil:
				t.Fatalf("Descending = nil, want %v", *tt.want)
			case tt.want != nil && *conf.Descending != *tt.want:
				t.Fatalf("Descending = %v, want %v", *conf.Descending, *tt.want)
			}
		})
	}
}
