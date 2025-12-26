package testutil

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartEmbeddedNATS(t *testing.T) {
	// Start embedded NATS server
	ns := StartEmbeddedNATS(t)

	// Verify server is running
	require.NotNil(t, ns)
	assert.True(t, ns.ReadyForConnections(1*time.Second))

	// Connect a client to verify server is accessible
	nc, err := nats.Connect(ns.ClientURL())
	require.NoError(t, err)
	defer nc.Close()

	// Verify basic pub/sub works
	subject := "test.subject"
	received := make(chan string, 1)

	// Subscribe
	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		received <- string(msg.Data)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Ensure subscription is active
	require.NoError(t, nc.Flush())

	// Publish
	testData := "hello nats"
	err = nc.Publish(subject, []byte(testData))
	require.NoError(t, err)

	// Wait for message
	select {
	case data := <-received:
		assert.Equal(t, testData, data)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}
