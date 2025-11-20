// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package common

import (
	"fmt"
	"time"
)

type Retrier struct {
	MaxRetry      uint
	Interval      uint
	Sleep         time.Duration
	ErrorCounters uint
}

func NewRetrier(maxRetry uint) (*Retrier, error) {
	if maxRetry == 0 {
		return nil, fmt.Errorf("maxRetry should be non-zero value")
	}
	return &Retrier{MaxRetry: maxRetry, Interval: 1, Sleep: time.Second}, nil
}

func (r Retrier) Wait() {
	time.Sleep(time.Duration(r.Interval) * r.Sleep)
}
