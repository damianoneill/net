package netconf

import (
	assert "github.com/stretchr/testify/require"
	"testing"
)


func TestMultipleSessions(t *testing.T) {

	ts := NewTestNetconfServer(t)
	assert.Nil(t, ts.LastReq(), "No requests should have been executed")

	ncs := newNCClientSession(t, ts)

	reply, err := ncs.Execute(Request(`<get><response/></get>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")

	ncs.Close()

	ncs = newNCClientSession(t, ts)
	defer ncs.Close()

	reply, err = ncs.Execute(Request(`<get><response/></get>`))
	assert.NoError(t, err, "Not expecting exec to fail")
	assert.NotNil(t, reply, "Reply should be non-nil")

}
