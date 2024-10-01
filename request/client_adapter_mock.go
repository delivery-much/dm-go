package request

import "github.com/delivery-much/mock-helper/mock"

type adapterMock struct {
	mock.Mock
}

func NewAdapterMock() *adapterMock {
	return &adapterMock{mock.NewMock()}
}

func (m *adapterMock) Adapt(c *Client) (i httpClientInterface) {
	res := m.GetResponseAndRegister("Adapt", c)
	if res.IsEmpty() {
		return
	}

	return res.Get(0).(httpClientInterface)
}
