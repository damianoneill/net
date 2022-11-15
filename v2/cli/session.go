package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/pkg/errors"

	"github.com/imdario/mergo"
)

// Session defines the API exposed by an SSH client.
type Session interface {
	// Send writes the supplied value to the server and returns the response.
	// The behaviour can be modified by opts - see SendOption variants below.
	Send(value string, opts ...SendOption) (string, error)
	io.Closer
}

// SendOption implements options for configuring Send behaviour.
type SendOption func(*SendConfig)

// WaitFor defines the string that indicates the end of the response to the send.
// Defaults to the current prompt.
func WaitFor(sentinel string) SendOption {
	return func(c *SendConfig) {
		c.responseSentinel = sentinel
	}
}

// NoNewline suppresses the newline that is by default appended to the Send string.
func NoNewline() SendOption {
	return func(c *SendConfig) {
		c.suppressNewline = true
	}
}

// ResetPrompt resets the current session prompt to the last unterminated line of response.
func ResetPrompt() SendOption {
	return func(c *SendConfig) {
		c.resetPrompt = true
	}
}

// NoWait indicates the Send should not wait for a response.
func NoWait() SendOption {
	return func(c *SendConfig) {
		c.noResponse = true
	}
}

// SendConfig defines properties controlling Send behaviour.
type SendConfig struct {
	suppressNewline  bool
	resetPrompt      bool
	noResponse       bool
	responseSentinel string
}

type SessionImpl struct {
	cfg   *SessionConfig
	tport SSHTransport
	// promptPattern defines the regex used to determine the end of a response.
	promptPattern *regexp.Regexp
	// Used to queue the inputs received from the server.
	inputs chan []byte
}

// NewCliSession establishes a client connection to a cli session running on the server associated with the supplied
// transport.
func NewCliSession(ctx context.Context, tport SSHTransport, cfg *SessionConfig) (s *SessionImpl, err error) {
	// Use supplied config, but apply any defaults to unspecified values.
	resolvedConfig := *cfg
	_ = mergo.Merge(&resolvedConfig, DefaultConfig)

	// If caller has specified a specific prompt pattern, check it's valid.
	var pattern *regexp.Regexp
	if resolvedConfig.pattern != "" {
		pattern, err = regexp.Compile(resolvedConfig.pattern)
		if err != nil {
			return nil, errors.Wrap(err, "invalid prompt pattern")
		}
	}

	sess := &SessionImpl{cfg: &resolvedConfig, tport: tport, inputs: make(chan []byte), promptPattern: pattern}

	// Launch the reader to capture input from the server.
	sess.launchReader()

	// Capture the cli prompt from the new session.
	if resolvedConfig.autoDetect {
		err = sess.capturePrompt()
	} else if pattern != nil {
		// Swallow the prompt value provided by the user.
		_, err = sess.readUntilValue(pattern)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to capture cli prompt")
	}

	// Execute any initial commands, ignoring any response values.
	for _, cmd := range sess.cfg.initCmds {
		_, err = sess.Send(cmd)
		if err != nil {
			return nil, errors.Wrap(err, "failed to execute initial command "+cmd)
		}
	}

	return sess, nil
}

// Captures the cli prompt.
// We keep reading until a read times out.
// Then we use the content after the last newline.
func (s *SessionImpl) capturePrompt() error {
	b, err := s.readUntilTimeout()
	if err != nil {
		return err
	}
	pbytes := b[bytes.LastIndex(b, []byte("\n"))+1:]
	s.promptPattern = regexp.MustCompile(regexp.QuoteMeta(string(pbytes)))
	return nil
}

// Keep reading input from the server, until a read times out.
func (s *SessionImpl) readUntilTimeout() ([]byte, error) {
	output := new(bytes.Buffer)
	for {
		select {
		case rd := <-s.inputs:
			if rd == nil {
				return nil, io.EOF
			}
			_, _ = output.Write(rd)
		case <-time.After(s.cfg.readTimeout):
			return output.Bytes(), nil
		}
	}
}

func (s *SessionImpl) Send(output string, opts ...SendOption) (string, error) {
	config := &SendConfig{}
	for _, opt := range opts {
		opt(config)
	}

	// If a response is expected, check that a prompt has been defined or the WaitFor option has been specified.
	if !config.noResponse && s.promptPattern == nil && config.responseSentinel == "" {
		return "", fmt.Errorf("need to specify WaitFor if cli prompt is not defined")
	}

	// If the caller has specified a "WaitFor" value - check it's a valid regex.
	var sentinel *regexp.Regexp
	var err error
	if config.responseSentinel != "" {
		sentinel, err = regexp.Compile(config.responseSentinel)
		if err != nil {
			return "", errors.Wrap(err, "invalid WaitFor value")
		}
	}

	// Write any output to the server.
	if len(output) > 0 {
		if !config.suppressNewline {
			output += "\n"
		}
		_, err = s.tport.Write([]byte(output))
		if err != nil {
			return "", errors.Wrap(err, "failed to send command")
		}
	}

	// Capture the response, unless none is expected.
	if config.noResponse {
		return "", nil
	}

	// If the output is expected to change the prompt value, capture the new prompt.
	if config.resetPrompt {
		return "", s.capturePrompt()
	}

	// Capture any input up to but not including the prompt.
	if sentinel == nil {
		sentinel = s.promptPattern
	}
	return s.readUntilValue(sentinel)
}

func (s *SessionImpl) Close() error {
	return s.tport.Close()
}

// readUntilValue reads until the specified regex is found and returns the read data.
func (s *SessionImpl) readUntilValue(sentinel *regexp.Regexp) (string, error) {
	output := new(bytes.Buffer)
	for {
		b := <-s.inputs
		if b == nil {
			return "", io.EOF
		}

		output.Write(b)
		tempSlice := bytes.ReplaceAll(output.Bytes(), []byte("\r\n"), []byte("\n"))
		tempSlice = bytes.ReplaceAll(tempSlice, []byte("\r"), []byte("\n"))
		lastNl := bytes.LastIndex(tempSlice, []byte("\n"))
		lastLine := tempSlice
		if lastNl >= 0 {
			lastLine = tempSlice[lastNl+1:]
		} else {
			lastNl = 0
		}
		if sentinel.Match(lastLine) {
			return string(tempSlice[0:lastNl]), nil
		}
	}
}

func (s *SessionImpl) launchReader() {
	go func() {
		defer close(s.inputs)
		for {
			const bufLength = 10000
			stdoutBuf := make([]byte, bufLength)
			byteCount, err := s.tport.Read(stdoutBuf)
			if err != nil {
				return
			}
			s.inputs <- stdoutBuf[:byteCount]
		}
	}()
}
