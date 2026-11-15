package config

import (
	"testing"

	"github.com/BurntSushi/toml"
)

func TestCategoryConfigDescending(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{input: "", want: false},
		{input: "descending = true", want: true},
		{input: "descending = false", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var conf CategoryConfig
			if _, err := toml.Decode(tt.input, &conf); err != nil {
				t.Fatal(err)
			}
			if conf.Descending != tt.want {
				t.Errorf("Descending = %v, want %v", conf.Descending, tt.want)
			}
		})
	}
}
