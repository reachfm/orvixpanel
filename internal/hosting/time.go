package hosting

import "time"

// timeNowUTC is the clock for release-name generation. Pulled
// out so tests can freeze it.
var timeNowUTC = func() time.Time { return time.Now().UTC() }
