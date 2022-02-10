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
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/gravitational/teleport/lib/defaults"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/srv/db/common"
	"github.com/gravitational/teleport/lib/srv/db/common/role"
	"github.com/gravitational/trace"
)

// List of all commands that Teleport handles in a special way.
const (
	helloCmd        = "hello"
	authCmd         = "auth"
	subscribeCmd    = "subscribe"
	psubscribeCmd   = "psubscribe"
	punsubscribeCmd = "punsubscribe"
	ssubscribeCmd   = "ssubscribe"
	sunsubscribeCmd = "sunsubscribe"
)

const defaultUsername = "default"

// processCmd processes commands received from connected client. Most commands are just passed to Redis instance,
// but some require special actions:
//  * Redis 7.0+ commands are rejected as at the moment of writing Redis 7.0 hasn't been released and go-redis doesn't support it.
//  * RESP3 commands are rejected as Teleport currently doesn't support this version of protocol.
//  * Subscribe related commands created a new DB connection as they change Redis request-response model to Pub/Sub.
func (e *Engine) processCmd(ctx context.Context, redisClient redis.UniversalClient, cmd *redis.Cmd) error {
	switch strings.ToLower(cmd.Name()) {
	case helloCmd:
		return trace.NotImplemented("RESP3 is not supported")
	case authCmd:
		if err := e.processAuth(ctx, redisClient, cmd); err != nil {
			return trace.Wrap(err)
		}
		return nil
	case subscribeCmd:
		e.Audit.OnQuery(e.Context, e.sessionCtx, common.Query{Query: cmd.String()})
		if err := e.subscribeCmd(ctx, redisClient.Subscribe, cmd); err != nil {
			return trace.Wrap(err)
		}
		return nil
	case psubscribeCmd:
		e.Audit.OnQuery(e.Context, e.sessionCtx, common.Query{Query: cmd.String()})
		if err := e.subscribeCmd(ctx, redisClient.PSubscribe, cmd); err != nil {
			return trace.Wrap(err)
		}
		return nil
	case punsubscribeCmd:
		// TODO(jakub): go-redis doesn't expose any API for this command. Investigate alternative options.
		return trace.NotImplemented("PUNSUBSCRIBE is not supported by Teleport")
	case ssubscribeCmd, sunsubscribeCmd:
		// go-redis doesn't support Redis 7+ commands yet.
		return trace.NotImplemented("Redis 7.0+ is not yet supported")
	default:
		e.Audit.OnQuery(e.Context, e.sessionCtx, common.Query{Query: cmd.String()})

		clusterCmd, err := processClusterCmd(ctx, redisClient, cmd)
		if err != nil || clusterCmd != nil {
			if clusterCmd != nil {
				cmd = clusterCmd
			}
			return err
		}

		// Here the command is sent to the DB.
		return redisClient.Process(ctx, cmd)
	}
}

type redisSubscribeFn = func(ctx context.Context, channels ...string) *redis.PubSub

// subscribeCmd handles subscription in Redis. A new connection is created and maintained to read published messages
// from Redis and send them back to the client.
func (e *Engine) subscribeCmd(ctx context.Context, subscribeFn redisSubscribeFn, cmd *redis.Cmd) error {
	if len(cmd.Args()) < 2 {
		return redis.RedisError("invalid command")
	}

	args := make([]string, 0, len(cmd.Args()))
	for _, a := range cmd.Args()[1:] {
		args = append(args, a.(string))
	}
	pubSub := subscribeFn(ctx, args...)
	defer func() {
		if err := pubSub.Close(); err != nil {
			e.Log.Errorf("failed to close Redis Pub/Sub connection: %v", err)
		}
	}()

	return e.processPubSub(ctx, pubSub)
}

// processPubSub reads messages from Redis and forward them to connected client.
func (e *Engine) processPubSub(ctx context.Context, pubSub *redis.PubSub) error {
	for {
		msg, err := pubSub.Receive(ctx)
		if err != nil {
			return err
		}

		switch msg := msg.(type) {
		case *redis.Subscription:
			if err := e.sendToClient([]interface{}{msg.Kind, msg.Channel, msg.Count}); err != nil {
				return err
			}
		case *redis.Pong:
			if err := e.sendToClient([]interface{}{msg.Payload}); err != nil {
				return err
			}
		case *redis.Message:
			var payloadResp interface{}
			if len(msg.PayloadSlice) > 0 {
				payloadResp = []interface{}{"message", msg.Channel, msg.PayloadSlice}
			} else {
				payloadResp = []interface{}{"message", msg.Channel, msg.Payload}
			}

			if err := e.sendToClient(payloadResp); err != nil {
				return err
			}
		default:
			err := fmt.Errorf("redis: unknown message: %T", msg)
			return err
		}
	}
}

// processAuth runs RBAC check on Redis AUTH command if command contains username. Command containing only password
// is passed to Redis. Commands with incorrect number of arguments are rejected and an error is returned.
func (e *Engine) processAuth(ctx context.Context, redisClient redis.UniversalClient, cmd *redis.Cmd) error {
	// AUTH command may contain only password or login and password. Depends on the version we need to make sure
	// that the user has permission to connect as the provided db user.
	// ref: https://redis.io/commands/auth
	switch len(cmd.Args()) {
	case 1:
		// Passed AUTH command without any arguments. Mimic Redis error response.
		return trace.BadParameter("wrong number of arguments for 'auth' command")
	case 2:
		e.Audit.OnQuery(e.Context, e.sessionCtx, common.Query{Query: "AUTH ****"})
		// Old Redis command. Password is the only argument here. Pass to Redis to validate.
		// ex. AUTH my-secret-password

		// Redis sets "default" as a default username. Here we need to check if the "implicit" username
		// matches the one provided as teleport db-user.
		// ref: https://redis.io/commands/auth
		if e.sessionCtx.DatabaseUser != defaultUsername {
			return trace.AccessDenied("failed to authenticate as the default user. " +
				"Please provide the db username when connection to Redis")
		}

		err := e.sessionCtx.Checker.CheckAccess(e.sessionCtx.Database,
			services.AccessMFAParams{Verified: true},
			role.DatabaseRoleMatchers(
				defaults.ProtocolRedis,
				e.sessionCtx.DatabaseUser,
				e.sessionCtx.DatabaseName,
			)...)
		if err != nil {
			return trace.Wrap(err)
		}

		return redisClient.Process(ctx, cmd)
	case 3:
		// Redis 6 version that contains username and password. Check the username against our RBAC before sending to Redis.
		// ex. AUTH bob my-secret-password
		dbUser, ok := cmd.Args()[1].(string)
		if !ok {
			return trace.BadParameter("username has a wrong type, expected string")
		}

		if dbUser != e.sessionCtx.DatabaseUser {
			return trace.AccessDenied("failed to authenticate as %s user. "+
				"Please provide a correct db username when connection to Redis", dbUser)
		}

		err := e.sessionCtx.Checker.CheckAccess(e.sessionCtx.Database,
			services.AccessMFAParams{Verified: true},
			role.DatabaseRoleMatchers(
				defaults.ProtocolRedis,
				e.sessionCtx.DatabaseUser,
				e.sessionCtx.DatabaseName,
			)...)
		if err != nil {
			return trace.Wrap(err)
		}

		e.Audit.OnQuery(e.Context, e.sessionCtx, common.Query{Query: fmt.Sprintf("AUTH %s ****", dbUser)})

		return redisClient.Process(ctx, cmd)
	default:
		// Redis returns "syntax error" if AUTH has more than 2 arguments.
		return trace.BadParameter("syntax error")
	}
}

// writeCmd writes Redis commands passed as vals to Redis wire form.
// Most types is covered by go-redis implemented WriteArg() function. Types override by this function are:
// * Redis errors and Go error: go-redis returns a "human-readable" string instead of RESP compatible error message
// * integers: go-redis converts them to string, which is not always what we want to.
// * slices: arrays are recursively converted to RESP responses.
func writeCmd(wr *redis.Writer, vals interface{}) error {
	switch val := vals.(type) {
	case redis.Error:
		if val == redis.Nil {
			// go-redis returns nil value as errors, but Redis Wire protocol decodes them differently.
			// Note: RESP3 has different sequence for nil, current nil is RESP2 compatible as the rest
			// of our implementation.
			if _, err := wr.WriteString("$-1\r\n"); err != nil {
				return trace.Wrap(err)
			}
			return nil
		}

		if err := writeError(wr, "-", val); err != nil {
			return trace.Wrap(err)
		}
	case trace.Error:
		if err := writeError(wr, "-ERR Teleport: ", val); err != nil {
			return trace.Wrap(err)
		}
	case error:
		if err := writeError(wr, "-ERR ", val); err != nil {
			return trace.Wrap(err)
		}
	case int:
		if err := writeInteger(wr, int64(val)); err != nil {
			return trace.Wrap(err)
		}
	case int8:
		if err := writeInteger(wr, int64(val)); err != nil {
			return trace.Wrap(err)
		}
	case int16:
		if err := writeInteger(wr, int64(val)); err != nil {
			return trace.Wrap(err)
		}
	case int32:
		if err := writeInteger(wr, int64(val)); err != nil {
			return trace.Wrap(err)
		}
	case int64:
		if err := writeInteger(wr, val); err != nil {
			return trace.Wrap(err)
		}
	case []string:
		if err := writeStringSlice(wr, val); err != nil {
			return trace.Wrap(err)
		}
	case []interface{}:
		if err := writeSlice(wr, val); err != nil {
			return trace.Wrap(err)
		}
	case interface{}:
		err := wr.WriteArg(val)
		if err != nil {
			return trace.Wrap(err)
		}
	}

	return nil
}

// writeError converts Go error to Redis wire form.
func writeError(wr *redis.Writer, prefix string, val error) error {
	if _, err := wr.WriteString(prefix); err != nil {
		// Add error header specified in https://redis.io/topics/protocol#resp-errors
		// to follow the convention.
		return trace.Wrap(err)
	}

	if _, err := wr.WriteString(val.Error()); err != nil {
		return trace.Wrap(err)
	}

	if _, err := wr.Write([]byte("\r\n")); err != nil {
		return trace.Wrap(err)
	}

	return nil
}

// writeSlice converts []interface{} to Redis wire form.
func writeSlice(wr *redis.Writer, vals []interface{}) error {
	if err := wr.WriteByte(redis.ArrayReply); err != nil {
		return trace.Wrap(err)
	}
	n := len(vals)
	if err := wr.WriteLen(n); err != nil {
		return trace.Wrap(err)
	}

	for _, v0 := range vals {
		if err := writeCmd(wr, v0); err != nil {
			return trace.Wrap(err)
		}
	}

	return nil
}

// writeStringSlice converts []string to Redis wire form.
func writeStringSlice(wr *redis.Writer, vals []string) error {
	if err := wr.WriteByte(redis.ArrayReply); err != nil {
		return trace.Wrap(err)
	}
	n := len(vals)
	if err := wr.WriteLen(n); err != nil {
		return trace.Wrap(err)
	}

	for _, v0 := range vals {
		if err := writeCmd(wr, v0); err != nil {
			return trace.Wrap(err)
		}
	}

	return nil
}

// writeInteger converts integer to Redis wire form.
func writeInteger(wr *redis.Writer, val int64) error {
	if err := wr.WriteByte(redis.IntReply); err != nil {
		return trace.Wrap(err)
	}

	v := strconv.FormatInt(val, 10)
	if _, err := wr.WriteString(v); err != nil {
		return trace.Wrap(err)
	}

	if _, err := wr.Write([]byte("\r\n")); err != nil {
		return trace.Wrap(err)
	}
	return nil
}

func processClusterCmd(ctx context.Context, redisClient redis.UniversalClient, cmd *redis.Cmd) (*redis.Cmd, error) {
	client, ok := redisClient.(*redis.ClusterClient)
	if !ok {
		return nil, nil
	}

	switch cmdName := strings.ToLower(cmd.Name()); cmdName {
	case "multi", "exec", "watch":
		return nil, trace.NotImplemented("%s is not supported in the cluster mode", cmdName)
	case "dbsize":
		res := client.DBSize(ctx)
		cmd.SetVal(res.Val())

		return cmd, res.Err()
	case "scan":
		var resultsKeys []string
		var mtx sync.Mutex

		if err := client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			scanCmd := redis.NewScanCmd(ctx, client.Process, cmd.Args()...)
			err := client.Process(ctx, scanCmd)
			if err != nil {
				return trace.Wrap(err)
			}

			keys, _, err := scanCmd.Result()
			if err != nil {
				return trace.Wrap(err)
			}

			mtx.Lock()
			resultsKeys = append(resultsKeys, keys...)
			mtx.Unlock()

			return nil
		}); err != nil {
			return nil, trace.Wrap(err)
		}

		cmd.SetVal(resultsKeys)

		return cmd, nil
	case "keys":
		var resultsKeys []string
		var mtx sync.Mutex

		if err := client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			scanCmd := redis.NewStringSliceCmd(ctx, cmd.Args()...)
			err := client.Process(ctx, scanCmd)
			if err != nil {
				return trace.Wrap(err)
			}

			keys, err := scanCmd.Result()
			if err != nil {
				return trace.Wrap(err)
			}

			mtx.Lock()
			resultsKeys = append(resultsKeys, keys...)
			mtx.Unlock()

			return nil
		}); err != nil {
			return nil, trace.Wrap(err)
		}

		cmd.SetVal(resultsKeys)

		return cmd, nil
		// TODO(jakub) mset may work, but without consistency guaranty
	case "mget":
		if len(cmd.Args()) == 1 {
			return nil, trace.BadParameter("wrong number of arguments for 'script' command")
		}

		var resultsKeys []interface{}
		var mtx sync.Mutex

		keys := cmd.Args()[1:]
		for _, k := range keys {
			result := client.Get(ctx, k.(string))
			if result.Err() == redis.Nil {
				mtx.Lock()
				resultsKeys = append(resultsKeys, redis.Nil)
				mtx.Unlock()
				continue
			}

			if result.Err() != nil {
				return nil, trace.Wrap(result.Err())
			}

			mtx.Lock()
			resultsKeys = append(resultsKeys, result.Val())
			mtx.Unlock()
		}

		cmd.SetVal(resultsKeys)

		return cmd, nil
	case "flushall", "flushdb":
		if err := client.ForEachMaster(ctx, func(ctx context.Context, client *redis.Client) error {
			singleCmd := redis.NewCmd(ctx, cmd.Args()...)
			err := client.Process(ctx, singleCmd)
			if err != nil {
				return trace.Wrap(err)
			}

			cmd.SetVal(singleCmd.Val())

			return nil
		}); err != nil {
			return nil, trace.Wrap(err)
		}

		return cmd, nil
	}

	return nil, nil
}
