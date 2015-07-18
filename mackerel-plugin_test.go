package mackerelplugin

import (
	"bytes"
	"math"
	"testing"
	"time"
)

func TestCalcDiff(t *testing.T) {
	var mp MackerelPlugin

	val1 := 10.0
	val2 := 0.0
	now := time.Now()
	last := time.Unix(now.Unix()-10, 0)

	diff, err := mp.calcDiff(val1, now, val2, last)
	if diff != 60 {
		t.Errorf("calcDiff: %f should be %f", diff, 60.0)
	}
	if err != nil {
		t.Error("calcDiff causes an error")
	}
}

func TestCalcDiffWithUInt32WithReset(t *testing.T) {
	var mp MackerelPlugin

	val := uint32(10)
	now := time.Now()
	lastval := uint32(12345)
	last := time.Unix(now.Unix()-60, 0)

	diff, err := mp.calcDiffUint32(val, now, lastval, last, 10)
	if err != nil {
	} else {
		t.Errorf("calcDiffUint32 with counter reset should cause an error: %f", diff)
	}
}

func TestCalcDiffWithUInt32Overflow(t *testing.T) {
	var mp MackerelPlugin

	val := uint32(10)
	now := time.Now()
	lastval := math.MaxUint32 - uint32(10)
	last := time.Unix(now.Unix()-60, 0)

	diff, err := mp.calcDiffUint32(val, now, lastval, last, 10)
	if diff != 21.0 {
		t.Errorf("calcDiff: last: %d, now: %d, %f should be %f", val, lastval, diff, 21.0)
	}
	if err != nil {
		t.Error("calcDiff causes an error")
	}
}

func TestCalcDiffWithUInt64WithReset(t *testing.T) {
	var mp MackerelPlugin

	val := uint64(10)
	now := time.Now()
	lastval := uint64(12345)
	last := time.Unix(now.Unix()-60, 0)

	diff, err := mp.calcDiffUint64(val, now, lastval, last, 10)
	if err != nil {
	} else {
		t.Errorf("calcDiffUint64 with counter reset should cause an error: %f", diff)
	}
}

func TestCalcDiffWithUInt64Overflow(t *testing.T) {
	var mp MackerelPlugin

	val := uint64(10)
	now := time.Now()
	lastval := math.MaxUint64 - uint64(10)
	last := time.Unix(now.Unix()-60, 0)

	diff, err := mp.calcDiffUint64(val, now, lastval, last, 10)
	if diff != 21.0 {
		t.Errorf("calcDiff: last: %d, now: %d, %f should be %f", val, lastval, diff, 21.0)
	}
	if err != nil {
		t.Error("calcDiff causes an error")
	}
}

func TestPrintValueUint32(t *testing.T) {
	var mp MackerelPlugin
	s := new(bytes.Buffer)
	var now = time.Unix(1437227240, 0)
	mp.printValue(s, "test", uint32(10), now)

	expected := []byte("test\t10\t1437227240\n")

	if bytes.Compare(expected, s.Bytes()) != 0 {
		t.Fatalf("not matched, expected: %s, got: %s", expected, s)
	}
}

func TestPrintValueUint64(t *testing.T) {
	var mp MackerelPlugin
	s := new(bytes.Buffer)
	var now = time.Unix(1437227240, 0)
	mp.printValue(s, "test", uint64(10), now)

	expected := []byte("test\t10\t1437227240\n")

	if bytes.Compare(expected, s.Bytes()) != 0 {
		t.Fatalf("not matched, expected: %s, got: %s", expected, s)
	}
}

func TestPrintValueFloat64(t *testing.T) {
	var mp MackerelPlugin
	s := new(bytes.Buffer)
	var now = time.Unix(1437227240, 0)
	mp.printValue(s, "test", float64(10.0), now)

	expected := []byte("test\t10.000000\t1437227240\n")

	if bytes.Compare(expected, s.Bytes()) != 0 {
		t.Fatalf("not matched, expected: %s, got: %s", expected, s)
	}
}
