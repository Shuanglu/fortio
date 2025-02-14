// Copyright 2015 Michal Witkowski. All Rights Reserved.
// See LICENSE for licensing terms.

package dflag

import (
	"flag"
	"testing"
	"time"
)

func TestDynInt64_SetAndGet(t *testing.T) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	dynFlag := DynInt64(set, "some_int_1", 13371337, "Use it or lose it")
	assert.Equal(t, int64(13371337), dynFlag.Get(), "value must be default after create")
	err := set.Set("some_int_1", "77007700\n")
	assert.NoError(t, err, "setting value must succeed")
	assert.Equal(t, int64(77007700), dynFlag.Get(), "value must be set after update")
}

func TestDynInt64_IsMarkedDynamic(t *testing.T) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	DynInt64(set, "some_int_1", 13371337, "Use it or lose it")
	assert.True(t, IsFlagDynamic(set.Lookup("some_int_1")))
}

func TestDynInt64_Mutator(t *testing.T) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	dynFlag := DynInt64(set, "int_2x", 0, "...").WithValueMutator(func(v int64) int64 { return 2 * v })
	set.Set("int_2x", "  \t42\n")
	assert.Equal(t, int64(84), dynFlag.Get(), "value must be 2x after create")
}

func TestDynInt64_FiresValidators(t *testing.T) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	DynInt64(set, "some_int_1", 13371337, "Use it or lose it").WithValidator(ValidateDynInt64Range(0, 2000))

	assert.NoError(t, set.Set("some_int_1", "300"), "no error from validator when in range")
	assert.Error(t, set.Set("some_int_1", "2001"), "error from validator when value out of range")
}

func TestDynInt64_FiresNotifier(t *testing.T) {
	waitCh := make(chan bool, 1)
	notifier := func(oldVal int64, newVal int64) {
		assert.EqualValues(t, int64(13371337), oldVal, "old value in notify must match previous value")
		assert.EqualValues(t, int64(77007700), newVal, "new value in notify must match set value")
		waitCh <- true
	}

	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	DynInt64(set, "some_int_1", 13371337, "Use it or lose it").WithNotifier(notifier)
	set.Set("some_int_1", "77007700")
	select {
	case <-time.After(5 * time.Millisecond):
		assert.Fail(t, "failed to trigger notifier")
	case <-waitCh:
	}
}

func Benchmark_Int64_Dyn_Get(b *testing.B) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	value := DynInt64(set, "some_int_1", 13371337, "Use it or lose it")
	set.Set("some_int_1", "77007700")
	for i := 0; i < b.N; i++ {
		x := value.Get()
		x = x + 1 // nolint
	}
}

func Benchmark_Int64_Normal_Get(b *testing.B) {
	set := flag.NewFlagSet("foobar", flag.ContinueOnError)
	valPtr := set.Int64("some_int_1", 13371337, "Use it or lose it")
	set.Set("some_int_1", "77007700")
	for i := 0; i < b.N; i++ {
		x := *valPtr
		x = x + 1 // nolint
	}
}
