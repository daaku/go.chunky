package chunky_test

import (
	"bytes"
	"testing"

	"github.com/daaku/go.chunky"
)

type writer struct {
	f func(b []byte) (int, error)
}

func (w writer) Write(b []byte) (int, error) {
	return w.f(b)
}

func TestSmallFlush(t *testing.T) {
	data := []byte("hello")
	var realw bytes.Buffer
	chunkyw := &chunky.Writer{
		Writer:         &realw,
		MaxWriteLength: len(data) + 1,
	}
	i, err := chunkyw.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if i != len(data) {
		t.Fatalf("was expecting %d but got %d", len(data), i)
	}
	if realw.Len() != 0 {
		t.Fatal("was expecting no data yet")
	}
	if err := chunkyw.Mark(); err != nil {
		t.Fatal(err)
	}
	if err := chunkyw.Flush(); err != nil {
		t.Fatal(err)
	}
	if realw.Len() != len(data) {
		t.Fatalf("was expecting %d but got %d", len(data), realw.Len())
	}
}

func TestAutoChunking(t *testing.T) {
	chunks := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("foo"),
		[]byte("bar"),
		[]byte("baz"),
		[]byte("42"),
		[]byte("blesh"),
	}
	expected := [][]byte{
		chunks[0],
		chunks[1],
		chunks[2],
		chunks[3],
		bytes.Join([][]byte{chunks[4], chunks[5]}, []byte("")),
		chunks[6],
	}

	finished := make(chan bool)
	var n int
	var realw = writer{
		f: func(b []byte) (int, error) {
			if n == len(expected) {
				t.Fatal("got more write calls than expected")
			}
			if n == len(expected)-1 {
				defer close(finished)
			}
			if !bytes.Equal(b, expected[n]) {
				t.Fatalf(`did not find expected "%s" instead got "%s"`, expected[n], b)
			}
			n++
			return len(b), nil
		},
	}

	chunkyw := &chunky.Writer{
		Writer:         realw,
		MaxWriteLength: 5,
	}

	for _, chunk := range chunks {
		_, err := chunkyw.Write(chunk)
		if err != nil {
			t.Fatal(err)
		}
		if err := chunkyw.Mark(); err != nil {
			t.Fatal(err)
		}
	}

	if err := chunkyw.Flush(); err != nil {
		t.Fatal(err)
	}
	<-finished
}

func TestBiggerThanMax(t *testing.T) {
	data := []byte("hello")
	var realw bytes.Buffer
	chunkyw := &chunky.Writer{
		Writer:         &realw,
		MaxWriteLength: len(data) - 1,
	}
	i, err := chunkyw.Write(data)
	if err == nil {
		t.Fatal("was expecting an error")
	}
	if i != 0 {
		t.Fatalf("was expecting %d but got %d", 0, i)
	}
}

func TestFlushBeforeMark(t *testing.T) {
	data := []byte("hello")
	var realw bytes.Buffer
	chunkyw := &chunky.Writer{
		Writer:         &realw,
		MaxWriteLength: len(data) + 1,
	}
	_, err := chunkyw.Write(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := chunkyw.Flush(); err == nil {
		t.Fatal("was expecting an error")
	}
	if realw.Len() != 0 {
		t.Fatal("was expecting no data yet")
	}
}
