/*
Copyright 2015 Gravitational, Inc.

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

package utils

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"time"

	"github.com/gravitational/trace"
)

// CryptoRandomHex returns a hex-encoded random string generated
// with a crypto-strong pseudo-random generator. The length parameter
// controls how many random bytes are generated, and the returned
// hex string will be twice the length. An error is returned when
// fewer bytes were generated than length.
func CryptoRandomHex(length int) (string, error) {
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", trace.Wrap(err)
	}
	return hex.EncodeToString(randomBytes), nil
}

// RandomDuration returns a duration in a range [0, max)
func RandomDuration(max time.Duration) time.Duration {
	randomVal, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return max / 2
	}
	return time.Duration(randomVal.Int64())
}
