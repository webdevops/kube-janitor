package kube_janitor

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"fortio.org/duration"
)

var (
	janitorTimeFormats = []string{
		// preferred format
		time.RFC3339,

		// human format
		"2006-01-02 15:04:05 +07:00",
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05",

		// allowed formats
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339Nano,

		// least preferred format
		"2006-01-02",
	}
)

func (j *Janitor) parseTimestamp(val string) *time.Time {
	val = strings.TrimSpace(val)
	if val == "" || val == "0" {
		return nil
	}

	// parse as unix timestamp
	if unixTimestamp, err := strconv.ParseInt(val, 10, 64); err == nil && unixTimestamp > 0 {
		timestamp := time.Unix(unixTimestamp, 0)
		return &timestamp
	}

	for _, timeFormat := range janitorTimeFormats {
		if timestamp, parseErr := time.Parse(timeFormat, val); parseErr == nil && timestamp.Unix() > 0 {
			return &timestamp
		}
	}

	return nil
}

func (j *Janitor) checkExpiryDate(createdAt time.Time, expiry string) (parsedTime *time.Time, expired bool, err error) {
	expired = false

	// sanity checks
	expiry = strings.TrimSpace(expiry)
	if expiry == "" || expiry == "0" {
		return
	}

	// first: parse as unix timestamp
	if unixTimestamp, err := strconv.ParseInt(expiry, 10, 64); err == nil && unixTimestamp > 0 {
		expiryTime := time.Unix(unixTimestamp, 0)
		parsedTime = &expiryTime
	}

	// second: parse duration
	if !createdAt.IsZero() {
		if expiryDuration, err := duration.Parse(expiry); err == nil && expiryDuration.Seconds() > 1 {
			expiryTime := createdAt.Add(expiryDuration)
			parsedTime = &expiryTime
		}
	}

	// third: parse time
	if parsedTime == nil {
		for _, timeFormat := range janitorTimeFormats {
			if parseVal, parseErr := time.Parse(timeFormat, expiry); parseErr == nil && parseVal.Unix() > 0 {
				parsedTime = &parseVal
				break
			}
		}
	}

	// check if time could be parsed
	if parsedTime != nil {
		// check if parsed time is before NOW -> expired
		expired = parsedTime.Before(time.Now())
	} else {
		err = fmt.Errorf("unable to parse time '%s'", expiry)
	}

	return
}
