/*
Copyright 2022 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package alpnproxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/gravitational/teleport/lib/utils"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"
)

// IsConnectRequest returns true if the request is a HTTP CONNECT tunnel
// request.
//
// https://datatracker.ietf.org/doc/html/rfc7231#section-4.3.6
func IsConnectRequest(req *http.Request) bool {
	return req.Method == "CONNECT"
}

// ForwardProxyConfig is the config for forward proxy server.
type ForwardProxyConfig struct {
	// TunnelProtocol is the protocol of the requests being tunneled.
	TunnelProtocol string
	// Listener is the network listener.
	Listener net.Listener
	// Receivers is a list of receivers that may receive from this proxy.
	Receivers []ForwardProxyReceiver
	// DropUnwantedRequests drops the request if no receiver wants the request.
	// If false, forward proxy sends the unwanted request to original host.
	DropUnwantedRequests bool
	// Log is the logger.
	Log logrus.FieldLogger
	// CloseContext is the close context.
	CloseContext context.Context

	// InsecureSystemProxy allows insecure system proxy when forwarding
	// unwanted requests.
	InsecureSystemProxy bool
	// SystemProxyFunc is the function that determines the system proxy URL to
	// use for a given request URL.
	SystemProxyFunc func(reqURL *url.URL) (*url.URL, error)
}

// CheckAndSetDefaults checks and sets default config values.
func (c *ForwardProxyConfig) CheckAndSetDefaults() error {
	if c.TunnelProtocol == "" {
		c.TunnelProtocol = "https"
	}
	if c.Listener == nil {
		return trace.BadParameter("missing listener")
	}
	if c.CloseContext == nil {
		return trace.BadParameter("missing close context")
	}
	if c.Log == nil {
		c.Log = logrus.WithField(trace.Component, "fwdproxy")
	}
	if c.SystemProxyFunc == nil {
		c.SystemProxyFunc = httpproxy.FromEnvironment().ProxyFunc()
	}
	return nil
}

// ForwardProxy is a forward proxy that serves CONNECT tunnel requests.
type ForwardProxy struct {
	cfg ForwardProxyConfig
}

// NewForwardProxy creates a new forward proxy server.
func NewForwardProxy(cfg ForwardProxyConfig) (*ForwardProxy, error) {
	if err := cfg.CheckAndSetDefaults(); err != nil {
		return nil, trace.Wrap(err)
	}

	return &ForwardProxy{
		cfg: cfg,
	}, nil
}

// Start starts serving requests.
func (p *ForwardProxy) Start() error {
	err := http.Serve(p.cfg.Listener, p)
	if err != nil && !utils.IsOKNetworkError(err) {
		return trace.Wrap(err)
	}
	return nil
}

// Close closes the forward proxy.
func (p *ForwardProxy) Close() error {
	if err := utils.CloseListener(p.cfg.Listener); err != nil {
		return trace.Wrap(err)
	}
	return nil
}

// ServeHTTP serves HTTP requests. Implements http.Handler.
func (p *ForwardProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Clients usually send plain HTTP requests directly without CONNECT tunnel
	// when proxying HTTP requests. These requests are rejected as only
	// requests through CONNECT tunnel are allowed.
	if !IsConnectRequest(req) {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	clientConn := hijackClientConnection(rw)
	if clientConn == nil {
		p.cfg.Log.Errorf("Failed to hijack client connection.")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	for _, receiver := range p.cfg.Receivers {
		if receiver.Want(req) {
			err := p.forwardClientToHost(clientConn, receiver.Addr().String(), req)
			if err != nil {
				p.cfg.Log.WithError(err).Errorf("Failed to handle forward request for %q.", req.Host)
				p.writeHeader(clientConn, req.Proto, http.StatusInternalServerError)
			}
			return
		}
	}

	p.handleUnwantedRequest(clientConn, req)
}

// handleUnwantedRequest handles the request when no receiver wants it.
func (p *ForwardProxy) handleUnwantedRequest(clientConn net.Conn, req *http.Request) {
	if p.cfg.DropUnwantedRequests {
		p.cfg.Log.Debugf("Dropped forward request for %q.", req.Host)
		p.writeHeader(clientConn, req.Proto, http.StatusBadRequest)
		return
	}

	// Honors system proxy if exists.
	systemProxyURL, err := p.cfg.SystemProxyFunc(&url.URL{
		Host:   req.Host,
		Scheme: p.cfg.TunnelProtocol,
	})
	if err == nil && systemProxyURL != nil {
		err = p.forwardClientToSystemProxy(clientConn, systemProxyURL, req)
		if err != nil {
			p.cfg.Log.WithError(err).Errorf("Failed to forward request for %q through system proxy %v.", req.Host, systemProxyURL)
			p.writeHeader(clientConn, req.Proto, http.StatusInternalServerError)
		}
		return
	}

	// Forward to the original host.
	err = p.forwardClientToHost(clientConn, req.Host, req)
	if err != nil {
		p.cfg.Log.WithError(err).Errorf("Failed to forward request for %q.", req.Host)
		p.writeHeader(clientConn, req.Proto, http.StatusInternalServerError)
	}
}

// forwardClientToHost forwards client connection to provided host.
func (p *ForwardProxy) forwardClientToHost(clientConn net.Conn, host string, req *http.Request) error {
	serverConn, err := net.Dial("tcp", host)
	if err != nil {
		return trace.Wrap(err)
	}

	defer serverConn.Close()

	// Let client know we are ready for proxying.
	if err := writeHeaderToHijackedConnection(clientConn, req.Proto, http.StatusOK); err != nil {
		return trace.Wrap(err)
	}
	return p.startTunnel(clientConn, serverConn, req)
}

// forwardClientToSystemProxy forwards client connection to provided system proxy.
func (p *ForwardProxy) forwardClientToSystemProxy(clientConn net.Conn, systemProxyURL *url.URL, req *http.Request) error {
	var err error
	var serverConn net.Conn
	switch strings.ToLower(systemProxyURL.Scheme) {

	case "http":
		serverConn, err = net.Dial("tcp", systemProxyURL.Host)
		if err != nil {
			return trace.Wrap(err)
		}

	case "https":
		serverConn, err = tls.Dial("tcp", systemProxyURL.Host, &tls.Config{
			InsecureSkipVerify: p.cfg.InsecureSystemProxy,
		})
		if err != nil {
			return trace.Wrap(err)
		}

	default:
		return trace.BadParameter("unsupported system proxy %v", systemProxyURL)
	}

	defer serverConn.Close()
	p.cfg.Log.Debugf("Connected to system proxy %v.", systemProxyURL)

	// Send original CONNECT request to system proxy.
	connectRequestBytes, err := httputil.DumpRequest(req, true)
	if err != nil {
		return trace.Wrap(err)
	}
	if _, err = serverConn.Write(connectRequestBytes); err != nil {
		return trace.Wrap(err)
	}

	return p.startTunnel(clientConn, serverConn, req)
}

// startTunnel starts streaming between client and server.
func (p *ForwardProxy) startTunnel(clientConn, serverConn net.Conn, req *http.Request) error {
	p.cfg.Log.Debugf("Started forwarding request for %q", req.Host)
	defer p.cfg.Log.Debugf("Stopped forwarding request for %q", req.Host)

	// Force close connections when close context is done.
	go func() {
		<-p.cfg.CloseContext.Done()

		clientConn.Close()
		serverConn.Close()
	}()

	errsChan := make(chan error, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)
	stream := func(reader, writer net.Conn) {
		_, err := io.Copy(reader, writer)
		if err != nil && !utils.IsOKNetworkError(err) {
			errsChan <- err
		}
		if readerConn, ok := reader.(*net.TCPConn); ok {
			readerConn.CloseRead()
		}
		if writerConn, ok := writer.(*net.TCPConn); ok {
			writerConn.CloseWrite()
		}
		wg.Done()
	}
	go stream(clientConn, serverConn)
	go stream(serverConn, clientConn)
	wg.Wait()

	var errs []error
	for len(errsChan) > 0 {
		errs = append(errs, <-errsChan)
	}
	return trace.NewAggregate(errs...)
}

// writeHeader writes HTTP status to hijacked client connection, and log a
// message if failed.
func (p *ForwardProxy) writeHeader(conn net.Conn, protocol string, statusCode int) {
	err := writeHeaderToHijackedConnection(conn, protocol, statusCode)
	if err != nil && !utils.IsOKNetworkError(err) {
		p.cfg.Log.WithError(err).Errorf("Failed to write status code %d to client connection", statusCode)
	}
}

// hijackClientConnection hijacks client connection.
func hijackClientConnection(rw http.ResponseWriter) net.Conn {
	hijacker, ok := rw.(http.Hijacker)
	if !ok {
		return nil
	}

	clientConn, _, _ := hijacker.Hijack()
	return clientConn
}

// writeHeaderToHijackedConnection writes HTTP status to hijacked connection.
func writeHeaderToHijackedConnection(conn net.Conn, protocol string, statusCode int) error {
	formatted := fmt.Sprintf("%s %d %s\r\n\r\n", protocol, statusCode, http.StatusText(statusCode))
	_, err := conn.Write([]byte(formatted))
	return trace.Wrap(err)
}
