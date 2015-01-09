package lock

import (
	"github.com/funkygao/assert"
	"github.com/funkygao/fae/config"
	"testing"
	"time"
)

func TestLockBasic(t *testing.T) {
	cf := &config.ConfigLock{MaxItems: 10, Expires: 10 * time.Second}
	l := New(cf)
	k1 := "hello"
	k2 := "world"

	assert.Equal(t, true, l.Lock(k1))
	assert.Equal(t, false, l.Lock(k1))
	assert.Equal(t, true, l.Lock(k2))
	assert.Equal(t, false, l.Lock(k2))

	t.Logf("%+v", l.items)

	l.Unlock(k1)
	assert.Equal(t, true, l.Lock(k1))
	l.Unlock(k2)
	assert.Equal(t, true, l.Lock(k2))
}

func TestLockExpires(t *testing.T) {
	cf := &config.ConfigLock{MaxItems: 10, Expires: 1 * time.Second}
	l := New(cf)
	k := "hello"
	l.Lock(k)
	assert.Equal(t, false, l.Lock(k))
	time.Sleep(time.Second)
	assert.Equal(t, true, l.Lock(k))
}

func BenchmarkLockBasic(b *testing.B) {
	cf := &config.ConfigLock{MaxItems: 10, Expires: 10 * time.Second}
	l := New(cf)
	k := "haha"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		l.Lock(k)
		l.Unlock(k)
	}
}
