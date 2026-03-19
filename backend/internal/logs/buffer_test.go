package logs

import (
	"reflect"
	"testing"
)

func TestNewBufferUsesDefaultWhenNonPositiveMaxLines(t *testing.T) {
	buf := NewBuffer(0)
	if buf.max != defaultMaxLines {
		t.Fatalf("expected default max lines %d, got %d", defaultMaxLines, buf.max)
	}
}

func TestBufferKeepsLastNLines(t *testing.T) {
	buf := NewBuffer(2)
	if _, err := buf.Write([]byte("one\ntwo\nthree\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got := buf.Lines()
	want := []string{"two", "three"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lines: got %v want %v", got, want)
	}
}

func TestBufferExposesPartialLineUntilFlushed(t *testing.T) {
	buf := NewBuffer(5)
	if _, err := buf.Write([]byte("one\ntwo")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got := buf.Lines()
	want := []string{"one", "two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lines before flush: got %v want %v", got, want)
	}

	if _, err := buf.Write([]byte("\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	gotAfterFlush := buf.Lines()
	if !reflect.DeepEqual(gotAfterFlush, want) {
		t.Fatalf("unexpected lines after flush: got %v want %v", gotAfterFlush, want)
	}
}

func TestBufferIgnoresEmptyLines(t *testing.T) {
	buf := NewBuffer(3)
	if _, err := buf.Write([]byte("\n\nalpha\n\n")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	got := buf.Lines()
	want := []string{"alpha"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected lines: got %v want %v", got, want)
	}
}
