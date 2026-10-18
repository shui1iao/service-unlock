// Narrow Hulu Japan transport compatibility; no global HTTP/2 policy change.
// SPDX-License-Identifier: AGPL-3.0-only
package extra

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// net/http's HTTP/2 StreamError implements As to a matching external struct
// (also used for x/net/http2 compatibility). This avoids string matching and
// importing an internal package or adding/upgrading a module.
type huluStreamCode uint32
type huluStreamError struct {
	StreamID uint32
	Code     huluStreamCode
	Cause    error
}

func (e huluStreamError) Error() string {
	return fmt.Sprintf("HTTP/2 stream %d code %d", e.StreamID, e.Code)
}
func isHuluProtocolError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var stream huluStreamError
	if !errors.As(err, &stream) || stream.StreamID == 0 {
		return false
	}
	if errors.Is(stream.Cause, context.Canceled) || errors.Is(stream.Cause, context.DeadlineExceeded) {
		return false
	}
	// Only PROTOCOL_ERROR / INTERNAL_ERROR. Not REFUSED_STREAM, CANCEL,
	// ENHANCE_YOUR_CALM, generic network errors, HTTP status codes or strings.
	return stream.Code == 1 || stream.Code == 2
}
func (c *Client) huluHTTP1Client() (*http.Client, error) {
	tr, ok := c.HTTP.Transport.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("Hulu HTTP/1.1 fallback requires the existing standard transport")
	}
	clone := tr.Clone()
	clone.ForceAttemptHTTP2 = false
	clone.Protocols = new(http.Protocols)
	clone.Protocols.SetHTTP1(true)
	if clone.TLSClientConfig != nil {
		clone.TLSClientConfig.NextProtos = []string{"http/1.1"}
	}
	copy := *c.HTTP
	copy.Transport = clone
	// Timeout remains unchanged; the original outer request context supplies the
	// shared remaining budget, covering this attempt, redirects and response body.
	return &copy, nil
}
