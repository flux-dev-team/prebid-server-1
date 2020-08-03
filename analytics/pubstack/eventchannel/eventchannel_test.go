package eventchannel

import (
	"bytes"
	"compress/gzip"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"sync"
	"testing"
	"time"
)

var maxByteSize = int64(15)
var maxEventCount = int64(3)
var maxTime = 2 * time.Hour

func readGz(encoded bytes.Buffer) string {
	gr, _ := gzip.NewReader(bytes.NewBuffer(encoded.Bytes()))
	defer gr.Close()

	decoded, _ := ioutil.ReadAll(gr)
	return string(decoded)
}

func newSender(data *[]byte) Sender {
	mux := &sync.Mutex{}
	return func(payload []byte) error {
		mux.Lock()
		defer mux.Unlock()
		event := bytes.Buffer{}
		event.Write(payload)
		*data = append(*data, readGz(event)...)
		return nil
	}
}

func TestEventChannel_isBufferFull(t *testing.T) {

	send := func(_ []byte) error { return nil }

	eventChannel := NewEventChannel(send, maxByteSize, maxEventCount, maxTime)
	defer eventChannel.Close()

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))

	assert.Equal(t, eventChannel.isBufferFull(), false)

	eventChannel.buffer([]byte("three"))

	assert.Equal(t, eventChannel.isBufferFull(), true)

	eventChannel.reset()

	assert.Equal(t, eventChannel.isBufferFull(), false)

	eventChannel.buffer([]byte("big-event-abcdefghijklmnopqrstuvwxyz"))

	assert.Equal(t, eventChannel.isBufferFull(), true)

}

func TestEventChannel_reset(t *testing.T) {
	send := func(_ []byte) error { return nil }

	eventChannel := NewEventChannel(send, maxByteSize, maxEventCount, maxTime)
	defer eventChannel.Close()

	assert.Equal(t, eventChannel.metrics.eventCount, int64(0))
	assert.Equal(t, eventChannel.metrics.bufferSize, int64(0))

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))

	assert.NotEqual(t, eventChannel.metrics.eventCount, int64(0))
	assert.NotEqual(t, eventChannel.metrics.bufferSize, int64(0))

	eventChannel.reset()

	assert.Equal(t, eventChannel.buff.Len(), 0)
	assert.Equal(t, eventChannel.metrics.eventCount, int64(0))
	assert.Equal(t, eventChannel.metrics.bufferSize, int64(0))
}

func TestEventChannel_flush(t *testing.T) {
	data := make([]byte, 0)
	send := newSender(&data)

	eventChannel := NewEventChannel(send, maxByteSize, maxEventCount, maxTime)
	defer eventChannel.Close()

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))
	eventChannel.buffer([]byte("three"))
	eventChannel.flush()
	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, string(data), "onetwothree")
}

func TestEventChannel_close(t *testing.T) {
	data := make([]byte, 0)
	send := newSender(&data)

	eventChannel := NewEventChannel(send, 15000, 15000, 2*time.Hour)

	eventChannel.buffer([]byte("one"))
	eventChannel.buffer([]byte("two"))
	eventChannel.buffer([]byte("three"))
	eventChannel.Close()

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, string(data), "onetwothree")
}

func TestEventChannel_Push(t *testing.T) {
	data := make([]byte, 0)
	send := newSender(&data)

	eventChannel := NewEventChannel(send, 15000, 5, 5*time.Millisecond)
	defer eventChannel.Close()

	eventChannel.Push([]byte("one"))
	eventChannel.Push([]byte("two"))
	eventChannel.Push([]byte("three"))
	eventChannel.Push([]byte("four"))
	eventChannel.Push([]byte("five"))
	eventChannel.Push([]byte("six"))
	eventChannel.Push([]byte("seven"))

	time.Sleep(10 * time.Millisecond)

	assert.Equal(t, string(data), "onetwothreefourfivesixseven")

}
