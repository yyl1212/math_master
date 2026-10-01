package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testPassphrase = "only a public test passphrase"

func TestArgon2PHC(t *testing.T) {
	h := NewArgon2Hasher(rand.Reader)
	one, err := h.Hash(context.Background(), testPassphrase)
	if err != nil {
		t.Fatal("hash failed")
	}
	two, err := h.Hash(context.Background(), testPassphrase)
	if err != nil || one == two {
		t.Fatal("salt reuse")
	}
	parts := strings.Split(one, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=19456,t=2,p=1" {
		t.Fatal("unsupported PHC parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) != 16 {
		t.Fatal("salt boundary failed")
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) != 32 {
		t.Fatal("key boundary failed")
	}
	for _, tt := range []struct {
		password string
		want     bool
	}{{testPassphrase, true}, {"a different wrong test password", false}} {
		ok, err := h.Verify(context.Background(), one, tt.password)
		if err != nil || ok != tt.want {
			t.Fatal("password verification failed")
		}
	}
	_, err = NewArgon2Hasher(brokenRandom{}).Hash(context.Background(), testPassphrase)
	if err == nil {
		t.Fatal("failed salt entropy accepted")
	}
}
func TestPHCRejectsUnboundedParameters(t *testing.T) {
	var calls atomic.Int32
	h := newHasher(rand.Reader, func(password, salt []byte, time, memory uint32, threads uint8, keyLen uint32) []byte {
		calls.Add(1)
		return make([]byte, keyLen)
	})
	salt := base64.RawStdEncoding.EncodeToString(make([]byte, 16))
	key := base64.RawStdEncoding.EncodeToString(make([]byte, 32))
	valid := "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "$" + key
	bad := []string{
		strings.Replace(valid, "m=19456", "m=4294967295", 1), strings.Replace(valid, "m=19456", "m=19456,m=19456", 1), strings.Replace(valid, "argon2id", "argon2i", 1),
		strings.Replace(valid, "v=19", "v=16", 1), strings.Replace(valid, "t=2", "t=1000000000", 1), strings.Replace(valid, "p=1", "p=4", 1),
		"$argon2id$v=19$m=19456,t=2,p=1$!$" + key, "$argon2id$v=19$m=19456,t=2,p=1$" + salt + "=$" + key,
		"$argon2id$v=19$m=19456,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(make([]byte, 15)) + "$" + key,
		"$argon2id$v=19$m=19456,t=2,p=1$" + salt + "$" + base64.RawStdEncoding.EncodeToString(make([]byte, 31)), valid + "$extra", strings.Repeat("a", 100000),
	}
	for i, phc := range bad {
		if ok, err := h.Verify(context.Background(), phc, testPassphrase); err == nil || ok {
			t.Fatalf("malformed hash %d accepted", i)
		}
	}
	if calls.Load() != 0 {
		t.Fatal("malformed PHC reached derivation")
	}
}
func TestArgon2SlotsSurviveCancellation(t *testing.T) {
	entered := make(chan int, 5)
	gates := make([]chan struct{}, 5)
	for i := range gates {
		gates[i] = make(chan struct{})
		defer close(gates[i])
	}
	h := newHasher(rand.Reader, func(password, salt []byte, time, memory uint32, threads uint8, keyLen uint32) []byte {
		i := int(password[len(password)-1] - '0')
		entered <- i
		<-gates[i]
		return make([]byte, keyLen)
	})
	results := make([]chan error, 4)
	var cancelFirst context.CancelFunc
	for i := 0; i < 4; i++ {
		ctx, c := context.WithCancel(context.Background())
		defer c()
		if i == 0 {
			cancelFirst = c
		}
		results[i] = make(chan error, 1)
		go func(i int, ctx context.Context) {
			_, err := h.Hash(ctx, testPassphrase+strconv.Itoa(i))
			results[i] <- err
		}(i, ctx)
	}
	for i := 0; i < 4; i++ {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("derivation did not start")
		}
	}
	assertFull := func() {
		t.Helper()
		ctx, c := context.WithTimeout(context.Background(), time.Second)
		defer c()
		_, err := h.Hash(ctx, testPassphrase+"4")
		var limited *RateLimitError
		if !errors.As(err, &limited) {
			t.Fatal("fifth hash was not rejected immediately")
		}
	}
	assertFull()
	cancelFirst()
	assertFull()
	gates[0] <- struct{}{}
	if err := <-results[0]; !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled hash produced usable output")
	}
	fifth := make(chan error, 1)
	go func() { _, err := h.Hash(context.Background(), testPassphrase+"4"); fifth <- err }()
	select {
	case i := <-entered:
		if i != 4 {
			t.Fatal("unexpected computation")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("one released slot not reusable")
	}
	// Exactly three original computations remain while the fifth calculation occupies the freed slot.
	assertFull()
	gates[4] <- struct{}{}
	if <-fifth != nil {
		t.Fatal("reused slot failed")
	}
	for i := 1; i < 4; i++ {
		gates[i] <- struct{}{}
		if <-results[i] != nil {
			t.Fatal("active hash failed")
		}
	}
}

type gatedRandom struct {
	entered chan struct{}
	release chan struct{}
}

func (r gatedRandom) Read(p []byte) (int, error) {
	r.entered <- struct{}{}
	<-r.release
	return rand.Read(p)
}
func TestArgon2ProcessLimit(t *testing.T) {
	r := gatedRandom{make(chan struct{}, 4), make(chan struct{}, 4)}
	results := make(chan error, 4)
	for i := 0; i < 4; i++ {
		h := NewArgon2Hasher(r)
		go func() { _, err := h.Hash(context.Background(), testPassphrase); results <- err }()
	}
	for i := 0; i < 4; i++ {
		select {
		case <-r.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("process computations not started")
		}
	}
	_, err := NewArgon2Hasher(rand.Reader).Hash(context.Background(), testPassphrase)
	var limited *RateLimitError
	for i := 0; i < 4; i++ {
		r.release <- struct{}{}
	}
	for i := 0; i < 4; i++ {
		if <-results != nil {
			t.Fatal("held calculation failed")
		}
	}
	if !errors.As(err, &limited) {
		t.Fatal("additional hasher exceeded process computation budget")
	}
}

func BenchmarkArgon2Hash(b *testing.B) {
	h := NewArgon2Hasher(rand.Reader)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := h.Hash(context.Background(), testPassphrase); err != nil {
			b.Fatal("hash failed")
		}
	}
}
func BenchmarkArgon2Concurrent(b *testing.B) {
	h := NewArgon2Hasher(rand.Reader)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		var failed atomic.Bool
		for n := 0; n < 4; n++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := h.Hash(context.Background(), testPassphrase); err != nil {
					failed.Store(true)
				}
			}()
		}
		wg.Wait()
		if failed.Load() {
			b.Fatal("concurrent hash failed")
		}
	}
}

var _ = bytes.NewReader
