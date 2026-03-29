package logs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.level.String())
	}
}

func TestLevel_Values(t *testing.T) {
	assert.Equal(t, Level(0), LevelDebug)
	assert.Equal(t, Level(1), LevelInfo)
	assert.Equal(t, Level(2), LevelWarn)
	assert.Equal(t, Level(3), LevelError)
}

func TestLevel_String_Unknown(t *testing.T) {
	assert.Equal(t, "UNKNOWN", Level(-1).String())
	assert.Equal(t, "UNKNOWN", Level(4).String())
}
