package mongo

import (
	"encoding/json"
	"testing"
	"time"

	v5_0_0 "github.com/netdata/go.d.plugin/modules/mongodb/testdata/v5.0.0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMongo_Init(t *testing.T) {
	tests := map[string]struct {
		config  Config
		success bool
	}{
		"success on default config": {
			success: true,
			config:  New().Config,
		},
		"fails on unset 'address'": {
			success: true,
			config: Config{
				Uri:     "mongodb://localhost:27017",
				Timeout: 10,
			},
		},
		"fails on invalid port": {
			success: false,
			config: Config{
				Uri:     "",
				Timeout: 0,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			m := New()
			m.Config = test.config
			assert.Equal(t, test.success, m.Init())
		})
	}
}

func TestMongo_Charts(t *testing.T) {
	m := New()
	require.True(t, m.Init())
	assert.NotNil(t, m.Charts())
}

func TestMongo_ChartsOptional(t *testing.T) {
	m := New()
	require.True(t, m.Init())
	assert.NotNil(t, m.Charts())
}

func TestMongo_initMongoClient_uri(t *testing.T) {
	m := New()
	m.Config.Uri = "mongodb://user:pass@localhost:27017"
	assert.True(t, m.Init())
}

func TestMongo_CheckFail(t *testing.T) {
	m := New()
	m.Config.Timeout = 0
	assert.False(t, m.Check())
}

func TestMongo_Success(t *testing.T) {
	m := New()
	m.Config.Timeout = 1
	m.Config.Uri = ""
	obj := &mockMongo{response: v5_0_0.ServerStatus}
	m.mongoCollector = obj
	assert.True(t, m.Check())
}

func TestMongo_Collect(t *testing.T) {
	m := New()
	m.Uri = ""
	ms := m.Collect()
	assert.Len(t, ms, 0)
}

func TestMongo_Incomplete(t *testing.T) {
	m := New()
	m.mongoCollector = &mockMongo{}
	ms := m.Collect()
	assert.Len(t, ms, 0)
}

func TestMongo_Cleanup(t *testing.T) {
	m := New()
	connector := &mockMongo{}
	m.mongoCollector = connector
	m.Cleanup()
	assert.True(t, connector.closeCalled)
}

type mockMongo struct {
	connector
	response    string
	closeCalled bool
}

func (m *mockMongo) initClient(_ string, _ time.Duration) error {
	return nil
}

func (m *mockMongo) serverStatus() (*serverStatus, error) {
	var status serverStatus
	err := json.Unmarshal([]byte(m.response), &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

func (m *mockMongo) close() error {
	m.closeCalled = true
	return nil
}