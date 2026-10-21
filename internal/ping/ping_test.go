package ping

import (
	"testing"
	"time"
)

func TestMeasureLayeredUnreachableIsError(t *testing.T) {
	// .invalid never resolves (RFC 2606), so every layer fails.
	res, err := MeasureLayered("no-such-host.invalid", 1, time.Second)
	if err == nil {
		t.Fatalf("expected an error, got mean %.2f ms", res.MeanMS)
	}
}
