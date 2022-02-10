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

package redis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/go-redis/redis/v8"
	"github.com/gravitational/teleport/lib/defaults"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/srv/db/common"
	"github.com/gravitational/teleport/lib/srv/db/common/role"
	"github.com/gravitational/teleport/lib/utils"
	"github.com/gravitational/trace"
)

func init() {
	common.RegisterEngine(newEngine, defaults.ProtocolRedis)
}

// newEngine create new Redis engine.
func newEngine(ec common.EngineConfig) common.Engine {
	return &Engine{
		EngineConfig: ec,
	}
}

// Engine implements common.Engine.
type Engine struct {
	// EngineConfig is the common database engine configuration.
	common.EngineConfig
	// proxyConn is a client connection.
	proxyConn net.Conn
	// clientReader is a go-redis wrapper for Redis client connection.
	clientReader *redis.Reader
	// sessionCtx is current session context.
	sessionCtx *common.Session
}

// InitializeConnection initializes the database connection.
func (e *Engine) InitializeConnection(clientConn net.Conn, sessionCtx *common.Session) error {
	e.proxyConn = clientConn
	e.clientReader = redis.NewReader(clientConn)
	e.sessionCtx = sessionCtx

	// Let client to not set the database user, as Redis uses default user "default" if a user
	// is not provided.
	if e.sessionCtx.DatabaseUser == "" {
		e.sessionCtx.DatabaseUser = defaultUsername
	}

	return nil
}

// authorizeConnection does authorization check for MongoDB connection about
// to be established.
func (e *Engine) authorizeConnection(ctx context.Context) error {
	ap, err := e.Auth.GetAuthPreference(ctx)
	if err != nil {
		return trace.Wrap(err)
	}
	mfaParams := services.AccessMFAParams{
		Verified:       e.sessionCtx.Identity.MFAVerified != "",
		AlwaysRequired: ap.GetRequireSessionMFA(),
	}

	dbRoleMatchers := role.DatabaseRoleMatchers(
		e.sessionCtx.Database.GetProtocol(),
		e.sessionCtx.DatabaseUser,
		e.sessionCtx.DatabaseName,
	)
	err = e.sessionCtx.Checker.CheckAccess(
		e.sessionCtx.Database,
		mfaParams,
		dbRoleMatchers...,
	)
	if err != nil {
		e.Audit.OnSessionStart(e.Context, e.sessionCtx, err)
		return trace.Wrap(err)
	}
	return nil
}

// SendError sends error message to connected client.
func (e *Engine) SendError(redisErr error) {
	if redisErr == nil || utils.IsOKNetworkError(redisErr) {
		return
	}

	if err := e.sendToClient(redisErr); err != nil {
		e.Log.Errorf("failed to send message to the client: %v", err)
		return
	}
}

// sendToClient sends a command to connected Redis client.
func (e *Engine) sendToClient(vals interface{}) error {
	buf := &bytes.Buffer{}
	wr := redis.NewWriter(buf)

	if err := writeCmd(wr, vals); err != nil {
		return trace.BadParameter("failed to convert error to a message: %v", err)
	}

	if _, err := e.proxyConn.Write(buf.Bytes()); err != nil {
		return trace.ConnectionProblem(err, "failed to send message to the client")
	}

	return nil
}

// HandleConnection is responsible for connecting to a Redis instance/cluster and
func (e *Engine) HandleConnection(ctx context.Context, sessionCtx *common.Session) error {
	// Check that the user has access to the database.
	err := e.authorizeConnection(ctx)
	if err != nil {
		return trace.Wrap(err)
	}

	tlsConfig, err := e.Auth.GetTLSConfig(ctx, sessionCtx)
	if err != nil {
		return trace.Wrap(err)
	}

	connectionOptions, err := ParseRedisURI(sessionCtx.Database.GetURI())
	if err != nil {
		return trace.BadParameter("Redis connection string is incorrect: %v", err)
	}

	var (
		redisConn      redis.UniversalClient
		connectionAddr = fmt.Sprintf("%s:%s", connectionOptions.address, connectionOptions.port)
	)

	// TODO(jakub): Investigate Redis Sentinel.
	switch connectionOptions.mode {
	case Standalone:
		redisConn = redis.NewClient(&redis.Options{
			Addr:      connectionAddr,
			TLSConfig: tlsConfig,
		})
	case Cluster:
		if sessionCtx.DatabaseName != "" {
			// ref: https://redis.io/commands/select
			return trace.BadParameter("Redis Cluster only supports database zero")
		}

		redisConn = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:     []string{connectionAddr},
			TLSConfig: tlsConfig,
		})
	default:
		// We've checked that while validating the config, but checking again can help with regression.
		return trace.BadParameter("incorrect connection mode %s", connectionOptions.mode)
	}

	defer func() {
		if err := redisConn.Close(); err != nil {
			e.Log.Errorf("failed to close Redis connection: %v", err)
		}
	}()

	e.Audit.OnSessionStart(e.Context, sessionCtx, nil)
	defer e.Audit.OnSessionEnd(e.Context, sessionCtx)

	if err := e.process(ctx, redisConn); err != nil {
		return trace.Wrap(err)
	}

	return nil
}

// readClientCmd reads commands from connected Redis client.
func (e *Engine) readClientCmd(ctx context.Context) (*redis.Cmd, error) {
	cmd := &redis.Cmd{}
	if err := cmd.ReadReply(e.clientReader); err != nil {
		return nil, trace.Wrap(err)
	}

	val, ok := cmd.Val().([]interface{})
	if !ok {
		return nil, trace.BadParameter("failed to cast Redis value to a slice")
	}

	return redis.NewCmd(ctx, val...), nil
}

// process is the main processing function for Redis. It reads commands passed from client and passes them to
// a Redis instance. It's also responsible for audit.
func (e *Engine) process(ctx context.Context, redisClient redis.UniversalClient) error {
	for {
		// Read commands from client.
		cmd, err := e.readClientCmd(ctx)
		if err != nil {
			return trace.Wrap(err)
		}

		// send valid commands to Redis instance/cluster.
		err = e.processCmd(ctx, redisClient, cmd)

		var vals interface{}
		if _, ok := err.(redis.Error); ok {
			vals = err
		} else if errors.Is(err, context.DeadlineExceeded) {
			// Do not return Deadline Exceeded to the client as it's not very self-explanatory.
			// Return "connection timeout" as this is what most likely happened.
			vals = trace.ConnectionProblem(err, "connection timeout")
		} else if err != nil {
			// Terminate connection only on connection errors. Errors that comes from Redis
			// should be propagated back to the client.
			if utils.IsOKNetworkError(err) {
				return trace.Wrap(err)
			}

			vals = err
		} else {
			vals, err = cmd.Result()
			if err != nil {
				return trace.Wrap(err)
			}
		}

		// Send response back to the client.
		if err := e.sendToClient(vals); err != nil {
			return trace.Wrap(err)
		}
	}
}
