package errc_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"golan-example/pkg/errc"
)

func TestCodeNew_UsesExample(t *testing.T) {
	err := errc.RepositoryExampleLoad.New("example repository is not implemented")

	assert.NotNil(t, err)
	assert.Equal(t, "repository.example.load", err.Code())
	assert.Equal(t, "example repository is not implemented", err.Message())
}

func TestCodeWrap_UsesExample(t *testing.T) {
	cause := errors.New("boom")
	err := errc.ServiceExampleRun.Wrap(cause, "run example service")

	assert.NotNil(t, err)
	assert.Equal(t, "service.example.run", err.Code())
	assert.Equal(t, "run example service", err.Message())
	assert.ErrorIs(t, err, cause)
}
