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
	// The elapsed time also includes the time from startTime to timer.Start,
	// which can take a few milliseconds (such as in JavaScript).
	tolassert.EqualTol(t, elapsed.Seconds(), timer.Total.Seconds(), 0.01)

	timer.Reset()
	assert.Zero(t, timer.St)
	assert.Zero(t, timer.Total)
	assert.Zero(t, timer.N)
}

func TestTimeAvg(t *testing.T) {
	var tm Time
	assert.Equal(t, time.Duration(0), tm.Avg())
	assert.Equal(t, time.Duration(0), tm.Stop(), "not started")
	tm.Total = 4 * time.Second
	tm.N = 2
	assert.Equal(t, 2*time.Second, tm.Avg())
	tm.ResetStart()
	assert.Equal(t, 0, tm.N)
	assert.False(t, tm.St.IsZero())
	d := tm.Stop()
	assert.GreaterOrEqual(t, d, time.Duration(0))
	assert.Equal(t, 1, tm.N)
	assert.Equal(t, d, tm.Avg())
}
