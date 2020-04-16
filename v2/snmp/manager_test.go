package snmp

import (
	"context"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestNoOpImplementations(t *testing.T) {
	m, err := NewFactory().NewManager(context.Background(), "localhost:161")
	assert.NoError(t, err)

	r, err := m.Get(context.Background(), []string{"1.3.6.1.2.1.2.2.1.1"})
	assert.Nil(t, r)
	assert.Nil(t, err)

	r, err = m.GetNext(context.Background(), []string{"1.3.6.1.2.1.2.2.1.1"})
	assert.Nil(t, r)
	assert.Nil(t, err)

	r, err = m.GetBulk(context.Background(), []string{"1.3.6.1.2.1.2.2.1.1"}, 5, 10)
	assert.Nil(t, r)
	assert.Nil(t, err)

	walker := func(p *PDU) error {
		return nil
	}
	err = m.GetWalk(context.Background(), "1.3.6.1.2.1.2.2.1.1", walker)
	assert.Nil(t, err)
}
