package timer

import (
	"testing"
	"time"

	"cogentcore.org/core/base/tolassert"
	"github.com/stretchr/testify/assert"
)

func TestTime(t *testing.T) {
	startTime := time.Now()
	timer := Time{}

	assert.Zero(t, timer.St)
	assert.Zero(t, timer.Total)
	assert.Zero(t, timer.N)

	timer.Start()
	assert.NotZero(t, timer.St)
	assert.Zero(t, timer.Total)
	assert.Zero(t, timer.N)

	time.Sleep(100 * time.Millisecond)
	timer.Stop()
	elapsed := time.Since(startTime)
	assert.Equal(t, 1, timer.N)
	tolassert.Equal(t, elapsed.Seconds(), timer.Total.Seconds())

	timer.Reset()
	assert.Zero(t, timer.St)
	assert.Zero(t, timer.Total)
	assert.Zero(t, timer.N)
}
