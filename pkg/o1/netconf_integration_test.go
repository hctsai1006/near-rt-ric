package o1

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	"github.com/hctsai1006/near-rt-ric/internal/config"
// 	"github.com/nemith/netconf"
// 	"github.com/sirupsen/logrus"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
// 	"golang.org/x/crypto/ssh"
// )

// func TestNetconfIntegration(t *testing.T) {
// 	log := logrus.New()

// 	o1Config := &config.O1Config{
// 		NETCONF: config.NETCONFConfig{
// 			ListenAddress: "127.0.0.1",
// 			Port:          8300,
// 		},
// 	}

// 	server, err := NewNetconfServer(o1Config, log, nil)
// 	require.NoError(t, err)

// 	err = server.Start(context.Background())
// 	require.NoError(t, err)
// 	defer server.Stop(context.Background())

// 	time.Sleep(100 * time.Millisecond) // Give the server a moment to start

// 	// Client
// 	sshConfig := &ssh.ClientConfig{
// 		User: "netconf",
// 		Auth: []ssh.AuthMethod{
// 			ssh.Password("netconf"),
// 		},
// 		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
// 	}
// 	session, err := netconf.DialSSH("127.0.0.1:8300", sshConfig)
// 	require.NoError(t, err)
// 	defer session.Close()

// 	t.Run("TestGetConfig", func(t *testing.T) {
// 		reply, err := session.GetConfig(netconf.Running)
// 		require.NoError(t, err)
// 		assert.Contains(t, reply.Data, "<data/>")
// 	})

// 	t.Run("TestEditConfig", func(t *testing.T) {
// 		config := `<config><test>value</test></config>`
// 		reply, err := session.EditConfig(netconf.Running, config)
// 		require.NoError(t, err)
// 		assert.True(t, reply.OK)
// 	})

// 	t.Run("TestLockUnlock", func(t *testing.T) {
// 		reply, err := session.Lock(netconf.Running)
// 		require.NoError(t, err)
// 		assert.True(t, reply.OK)

// 		reply, err = session.Unlock(netconf.Running)
// 		require.NoError(t, err)
// 		assert.True(t, reply.OK)
// 	})
// }
