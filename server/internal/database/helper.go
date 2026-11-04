package database

import "time"

func getUnixTimeStamp(t time.Time) int64 {
	return t.Unix()
}

func getTimeFromUnixTS(ts int64) time.Time {
	return time.Unix(ts, 0)
}
