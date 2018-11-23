package netconf

import (
	assert "github.com/stretchr/testify/require"
	"testing"
)


func TestMultipleSessions(t *testing.T) {

	ts := NewTestNetconfServer(t)

	ncs := newNCClientSession(t, ts)
	assert.Nil(t, ts.SessionHandler(ncs.ID()).LastReq(), "No requests should have been executed")

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
