/*
Copyright 2020 Gravitational, Inc.

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

package common

import (
	"context"
	"fmt"

	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/tlsca"
	"github.com/gravitational/trace"

	"github.com/sirupsen/logrus"
)

// Session combines parameters for a database connection session.
type Session struct {
	// ID is the unique session ID.
	ID string
	// ClusterName is the cluster the database service is a part of.
	ClusterName string
	// HostID is the id of this database server host.
	HostID string
	// Database is the database user is connecting to.
	Database types.Database
	// Identity is the identity of the connecting Teleport user.
	Identity tlsca.Identity
	// Checker is the access checker for the identity.
	Checker services.AccessChecker
	// DatabaseUser is the requested database user.
	DatabaseUser string
	// DatabaseName is the requested database name.
	DatabaseName string
	// StartupParameters define initial connection parameters such as date style.
	StartupParameters map[string]string
	// Log is the logger with session specific fields.
	Log logrus.FieldLogger
	// LockTargets is a list of lock targets applicable to this session.
	LockTargets []types.LockTarget
}

// String returns string representation of the session parameters.
func (c *Session) String() string {
	return fmt.Sprintf("db[%v] identity[%v] dbUser[%v] dbName[%v]",
		c.Database.GetName(), c.Identity.Username, c.DatabaseUser, c.DatabaseName)
}

// CreateTracker creates a new session tracker for the database session.
func (c *Session) CreateTracker(ctx context.Context, engineCfg EngineConfig) error {
	engineCfg.Log.Debug("Creating session tracker")
	initiator := &types.Participant{
		ID:   c.DatabaseUser,
		User: c.Identity.Username,
	}

	tracker, err := types.NewSessionTracker(types.SessionTrackerSpecV1{
		SessionID:    c.ID,
		Kind:         string(types.DatabaseSessionKind),
		State:        types.SessionState_SessionStateRunning,
		Hostname:     c.HostID,
		DatabaseName: c.DatabaseName,
		ClusterName:  c.ClusterName,
		Login:        "root",
		Participants: []types.Participant{*initiator},
		HostUser:     initiator.User,
	})
	if err != nil {
		return trace.Wrap(err)
	}

	err = engineCfg.AuthClient.UpsertSessionTracker(ctx, tracker)
	if err != nil {
		return trace.Wrap(err)
	}

	// Start go routine to push back session expiration until ctx is canceled (session ends).
	go func() {
		err = services.UpdateSessionTrackerExpiryLoop(ctx, engineCfg.AuthClient, c.ID, engineCfg.Clock)
		if err != nil {
			engineCfg.Log.WithError(err).Warningf("Failed to update session tracker expiration for session %v.", c.ID)
		}
	}()

	return nil
}
