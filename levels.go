package logrus_appinsights

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

type RequestData struct {
	Method       string
	Url          string
	Duration     time.Duration
	responseCode string
}

type DependencyData struct {
	Name           string
	dependencyType string
	target         string
	success        bool
	Data           string
	ResultCode     string
}

type ExceptionData struct {
	Exception interface{}
}

const (
	Dependency logrus.Level = iota + 1000
	Request
	Exception
)

const (
	DependencyKey = "dependency"
	RequestKey    = "Request"
	ExceptionKey  = "Exception"
)

var (
	KeyDoesntExist           = errors.New("key doesn't exist in Data")
	DependencyKeyDoesntExist = fmt.Errorf("%w: %s", KeyDoesntExist, DependencyKey)
	RequestKeyDoesntExist    = fmt.Errorf("%w: %s", KeyDoesntExist, RequestKey)
	ExceptionKeyDoesntExist  = fmt.Errorf("%w: %s", KeyDoesntExist, ExceptionKey)
)

var (
	AssertionFailure           = errors.New("assertion failure")
	DependencyAssertionFailure = fmt.Errorf("%w: %s", AssertionFailure, DependencyKey)
	RequestAssertionFailure    = fmt.Errorf("%w: %s", AssertionFailure, RequestKey)
	ExceptionAssertionFailure  = fmt.Errorf("%w: %s", AssertionFailure, ExceptionKey)
)
