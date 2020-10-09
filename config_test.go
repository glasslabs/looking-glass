package glass_test

import (
	"testing"

	glass "github.com/glasslabs/looking-glass"
	"github.com/stretchr/testify/assert"
)

func TestParseSecrets(t *testing.T) {
	tests := []struct {
		name    string
		in      []byte
		wantErr bool
		want    map[string]interface{}
	}{
		{
			name: "valid config",
			in:   []byte("test:\n  something: 1"),
			want: map[string]interface{}{"test": map[string]interface{}{"something": 1}},
		},
		{
			name:    "invalid config",
			in:      []byte("test: something: 1"),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := glass.ParseSecrets(test.in)

			if test.wantErr {
				assert.Error(t, err)
				return
			}
			if assert.NoError(t, err) {
				assert.Equal(t, test.want, got)
			}
		})
	}
}
