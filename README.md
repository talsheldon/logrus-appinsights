[![Build Status](https://travis-ci.org/jjcollinge/logrus-appinsights.svg?branch=master)](https://travis-ci.org/jjcollinge/logrus-appinsights)
[![Coverage Status](https://coveralls.io/repos/github/jjcollinge/logrus-appinsights/badge.svg?branch=master)](https://coveralls.io/github/jjcollinge/logrus-appinsights?branch=master)
[![codecov](https://codecov.io/gh/jjcollinge/logrus-appinsights/branch/master/graph/badge.svg)](https://codecov.io/gh/jjcollinge/logrus-appinsights)


# Application Insights Hook for Logrus <img src="http://i.imgur.com/hTeVwmJ.png" width="40" height="40" alt=":walrus:" class="emoji" title=":walrus:"/>

## Usage

See examples folder.

```go
package main

import (
	"time"

	"github.com/jjcollinge/logrus-appinsights"
	log "github.com/sirupsen/logrus"
)

func init() {
	hook, err := logrus_appinsights.New("my_client", logrus_appinsights.Config{
		InstrumentationKey: "instrumentation_key",
		MaxBatchSize:       10,              // optional
		MaxBatchInterval:   time.Second * 5, // optional
	})
	if err != nil || hook == nil {
		panic(err)
	}

	// set custom levels
	hook.SetLevels([]log.Level{
		log.PanicLevel,
		log.ErrorLevel,
	})

	// ignore fields
	hook.AddIgnore("private")
	log.AddHook(hook)
}

func main() {

	f := log.Fields{
		"field1":  "field1_value",
		"field2":  "field2_value",
		"private": "private_value",
	}

	// Send log to Application Insights
	for {
        log.WithFields(f).Log(log.ErrorLevel, "my message")
		time.Sleep(time.Second * 1)
	}
}
```