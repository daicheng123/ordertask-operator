package retry_util

import (
	"errors"
	"fmt"
	"time"
)

type ConditionFunc func() (bool, error)

type RetryError struct {
	times int
}

func (err *RetryError) Error() string {
	return fmt.Sprintf("still failing after %d retries", err.times)
}

func IsRetryFailure(err error) bool {
	_, ok := err.(*RetryError)
	return ok
}

func Retry(interval time.Duration, maxRetries int, f ConditionFunc) error {
	if maxRetries <= 0 {
		return errors.New("maxRetries should be > 0")
	}

	tick := time.NewTicker(interval)
	defer tick.Stop()

	for i := 0; i < maxRetries; i++ {
		ok, err := f()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if i == maxRetries {
			break
		}
		<-tick.C
	}
	return &RetryError{times: maxRetries}
}
