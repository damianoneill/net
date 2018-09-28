package netconf

// Request represents the body of a Netconf RPC request.
type Request string

// Response represents the response to a Netconf RPC request.
type Response struct{}

// ResponseHandler defines a callback function that will be invoked to handle a response to
// an asynchronous request.
type ResponseHandler func(res Response)

// Session represents a Netconf Session
type Session interface {
	Execute(req Request) (*Response, error)
	ExecuteAsync(req Request, resh ResponseHandler) error
}

type sesImpl struct {
	t Transport
}

// NewSession creates a new Netconf session, using the supplied Transport.
func NewSession(t Transport) (Session, error) {
	return &sesImpl{t: t}, nil
}

func (si *sesImpl) Execute(req Request) (*Response, error) {
	return nil, nil
}

func (si *sesImpl) ExecuteAsync(req Request, resh ResponseHandler) error {
	return nil
}
