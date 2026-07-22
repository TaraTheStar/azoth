// SPDX-License-Identifier: AGPL-3.0-or-later

package vertex

import "time"

// probeInterval is how often the disconnected Client re-pings its
// endpoint while waiting to recover. azoth/llm keeps the equivalent constant
// unexported for its OpenAIClient; this adapter carries its own copy. Value
// matches the OpenAIClient default: recovery-aware without being chatty.
const probeInterval = 5 * time.Second
