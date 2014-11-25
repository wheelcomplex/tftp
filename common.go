package tftp

import (
	"log"
)

type config interface {
	RetryCount() (int)
	Timeout() (int)
	Log() (*log.Logger)
}

const (
	DEFAULT_TIMEOUT = 5 // seconds
	DEFAULT_RETRY_COUNT = 3
)
