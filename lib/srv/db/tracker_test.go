/*
Copyright 2021 Gravitational, Inc.

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

package db

import (
	"context"
	"testing"

	"github.com/gravitational/teleport/api/types"
	libevents "github.com/gravitational/teleport/lib/events"
	"github.com/gravitational/teleport/lib/srv/db/redis"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TestSessionTrackerPostgres verifies session tracker functionality emitted for Postgres
// connections.
func TestSessionTrackerPostgres(t *testing.T) {
	ctx := context.Background()
	testCtx := setupTestContext(ctx, t, withSelfHostedPostgres("postgres"))
	go testCtx.startHandlingConnections()

	testCtx.createUserAndRole(ctx, t, "alice", "admin", []string{"postgres"}, []string{"postgres"})

	trackers, err := testCtx.authClient.GetActiveSessionTrackers(ctx)
	require.NoError(t, err)
	require.Empty(t, trackers)

	// Session tracker should be created for new connection
	psql, err := testCtx.postgresClient(ctx, "alice", "postgres", "postgres", "postgres")
	require.NoError(t, err)

	trackers, err = testCtx.authClient.GetActiveSessionTrackers(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(trackers))
	require.Equal(t, types.SessionState_SessionStateRunning, trackers[0].GetState())

	// Closing connection should trigger session tracker state to be terminated.
	err = psql.Close(ctx)
	require.NoError(t, err)

	trackers, err = testCtx.authClient.GetActiveSessionTrackers(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(trackers))
	require.Equal(t, types.SessionState_SessionStateTerminated, trackers[0].GetState())
}

// TestSessionTrackerMySQL verifies session tracker functionality emitted for MySQL
// connections.
func TestSessionTrackerMySQL(t *testing.T) {
	ctx := context.Background()
	testCtx := setupTestContext(ctx, t, withSelfHostedMySQL("mysql"))
	go testCtx.startHandlingConnections()

	testCtx.createUserAndRole(ctx, t, "alice", "admin", []string{"root"}, []string{types.Wildcard})

	// Access denied should trigger an unsuccessful session start event.
	_, err := testCtx.mysqlClient("alice", "mysql", "notroot")
	require.Error(t, err)
	requireEvent(t, testCtx, libevents.DatabaseSessionStartFailureCode)

	// Connect should trigger successful session start event.
	mysql, err := testCtx.mysqlClient("alice", "mysql", "root")
	require.NoError(t, err)
	requireEvent(t, testCtx, libevents.DatabaseSessionStartCode)

	// Simple query should trigger the query event.
	_, err = mysql.Execute("select 1")
	require.NoError(t, err)
	requireQueryEvent(t, testCtx, libevents.DatabaseSessionQueryCode, "select 1")

	// Closing connection should trigger session end event.
	err = mysql.Close()
	require.NoError(t, err)
	requireEvent(t, testCtx, libevents.DatabaseSessionEndCode)
}

// TestSessionTrackerMongo verifies session tracker functionality emitted for MongoDB
// connections.
func TestSessionTrackerMongo(t *testing.T) {
	ctx := context.Background()
	testCtx := setupTestContext(ctx, t, withSelfHostedMongo("mongo"))
	go testCtx.startHandlingConnections()

	testCtx.createUserAndRole(ctx, t, "alice", "admin", []string{"admin"}, []string{"admin"})

	// Access denied should trigger an unsuccessful session start event.
	_, err := testCtx.mongoClient(ctx, "alice", "mongo", "notadmin")
	require.Error(t, err)
	waitForEvent(t, testCtx, libevents.DatabaseSessionStartFailureCode)

	// Connect should trigger successful session start event.
	mongoClient, err := testCtx.mongoClient(ctx, "alice", "mongo", "admin")
	require.NoError(t, err)
	waitForEvent(t, testCtx, libevents.DatabaseSessionStartCode)

	// Find command in a database we don't have access to.
	_, err = mongoClient.Database("notadmin").Collection("test").Find(ctx, bson.M{})
	require.Error(t, err)
	waitForEvent(t, testCtx, libevents.DatabaseSessionQueryFailedCode)

	// Find command should trigger the query event.
	_, err = mongoClient.Database("admin").Collection("test").Find(ctx, bson.M{})
	require.NoError(t, err)
	waitForEvent(t, testCtx, libevents.DatabaseSessionQueryCode)

	// Closing connection should trigger session end event.
	err = mongoClient.Disconnect(ctx)
	require.NoError(t, err)
	waitForEvent(t, testCtx, libevents.DatabaseSessionEndCode)
}

func TestSessionTrackerRedis(t *testing.T) {
	ctx := context.Background()
	testCtx := setupTestContext(ctx, t, withSelfHostedRedis("redis"))
	go testCtx.startHandlingConnections()

	testCtx.createUserAndRole(ctx, t, "alice", "admin", []string{"admin"}, []string{types.Wildcard})

	t.Run("access denied", func(t *testing.T) {
		// Access denied should trigger an unsuccessful session start event.
		_, err := testCtx.redisClient(ctx, "alice", "redis", "notadmin")
		require.Error(t, err)
		waitForEvent(t, testCtx, libevents.DatabaseSessionStartFailureCode)
	})

	var redisClient *redis.Client

	t.Run("session starts event", func(t *testing.T) {
		// Connect should trigger successful session start event.
		var err error
		redisClient, err = testCtx.redisClient(ctx, "alice", "redis", "admin")
		require.NoError(t, err)
		waitForEvent(t, testCtx, libevents.DatabaseSessionStartCode)
	})

	t.Run("command sends", func(t *testing.T) {
		// SET should trigger Query event.
		err := redisClient.Set(ctx, "foo", "bar", 0).Err()
		require.NoError(t, err)
		waitForEvent(t, testCtx, libevents.DatabaseSessionQueryCode)
	})

	t.Run("session ends event", func(t *testing.T) {
		// Closing connection should trigger session end event.
		err := redisClient.Close()
		require.NoError(t, err)
		waitForEvent(t, testCtx, libevents.DatabaseSessionEndCode)
	})
}
