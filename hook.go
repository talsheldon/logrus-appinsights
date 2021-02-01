package logrus_appinsights

import (
	"code.cloudfoundry.org/clock"
	"encoding/json"
	"fmt"
	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/sirupsen/logrus"
)

var defaultLevels = []logrus.Level{
	logrus.PanicLevel,
	logrus.FatalLevel,
	logrus.ErrorLevel,
	logrus.WarnLevel,
	logrus.InfoLevel,
	logrus.DebugLevel,
	logrus.TraceLevel,
}

var levelMap = map[logrus.Level]contracts.SeverityLevel{
	logrus.PanicLevel: appinsights.Critical,
	logrus.FatalLevel: appinsights.Critical,
	logrus.ErrorLevel: appinsights.Error,
	logrus.WarnLevel:  appinsights.Warning,
	logrus.InfoLevel:  appinsights.Information,
	logrus.DebugLevel: appinsights.Verbose,
	logrus.TraceLevel: appinsights.Verbose,
}

// AppInsightsHook is a logrus hook for Application Insights
type AppInsightsHook struct {
	client appinsights.TelemetryClient

	async        bool
	levels       []logrus.Level
	ignoreFields map[string]struct{}
	filters      map[string]func(interface{}) interface{}
}

// New returns an initialised logrus hook for Application Insights
func New(name string, conf Config) (*AppInsightsHook, error) {
	if conf.InstrumentationKey == "" {
		return nil, fmt.Errorf("InstrumentationKey is required and missing from configuration")
	}
	telemetryConf := appinsights.NewTelemetryConfiguration(conf.InstrumentationKey)
	if conf.MaxBatchSize != 0 {
		telemetryConf.MaxBatchSize = conf.MaxBatchSize
	}
	if conf.MaxBatchInterval != 0 {
		telemetryConf.MaxBatchInterval = conf.MaxBatchInterval
	}
	if conf.EndpointUrl != "" {
		telemetryConf.EndpointUrl = conf.EndpointUrl
	}
	telemetryClient := appinsights.NewTelemetryClientFromConfig(telemetryConf)
	if name != "" {
		telemetryClient.Context().Tags.Cloud().SetRole(name)
	}
	return &AppInsightsHook{
		client:       telemetryClient,
		levels:       defaultLevels,
		ignoreFields: make(map[string]struct{}),
		filters:      make(map[string]func(interface{}) interface{}),
	}, nil
}

// NewWithAppInsightsConfig returns an initialised logrus hook for Application Insights
func NewWithAppInsightsConfig(name string, conf *appinsights.TelemetryConfiguration) (*AppInsightsHook, error) {
	if conf == nil {
		return nil, fmt.Errorf("Nil configuration provided")
	}
	if conf.InstrumentationKey == "" {
		return nil, fmt.Errorf("InstrumentationKey is required in configuration")
	}
	telemetryClient := appinsights.NewTelemetryClientFromConfig(conf)
	if name != "" {
		telemetryClient.Context().Tags.Cloud().SetRole(name)
	}
	return &AppInsightsHook{
		client:       telemetryClient,
		levels:       defaultLevels,
		ignoreFields: make(map[string]struct{}),
		filters:      make(map[string]func(interface{}) interface{}),
	}, nil
}

// Levels returns logging level to fire this hook.
func (hook *AppInsightsHook) Levels() []logrus.Level {
	return hook.levels
}

// SetLevels sets logging level to fire this hook.
func (hook *AppInsightsHook) SetLevels(levels []logrus.Level) {
	hook.levels = levels
}

// SetAsync sets async flag for sending logs asynchronously.
// If use this true, Fire() does not return error.
func (hook *AppInsightsHook) SetAsync(async bool) {
	hook.async = async
}

// AddIgnore adds field name to ignore.
func (hook *AppInsightsHook) AddIgnore(name string) {
	hook.ignoreFields[name] = struct{}{}
}

// AddFilter adds a custom filter function.
func (hook *AppInsightsHook) AddFilter(name string, fn func(interface{}) interface{}) {
	hook.filters[name] = fn
}

// Fire is invoked by logrus and sends log data to Application Insights.
func (hook *AppInsightsHook) Fire(entry *logrus.Entry) error {
	if !hook.async {
		return hook.fire(entry)
	}
	// async - fire and forget
	go hook.fire(entry)
	return nil
}

func (hook *AppInsightsHook) fire(entry *logrus.Entry) error {
	switch entry.Level {
	case Dependency:
		return hook.fireDependency(entry)
	case Request:
		return hook.fireRequest(entry)
	default:
	}
	trace, err := hook.buildTrace(entry)
	if err != nil {
		return err
	}
	hook.client.Track(trace)
	return nil
}

func (hook *AppInsightsHook) fireDependency(entry *logrus.Entry) error {
	val, ok := entry.Data[DependencyKey]
	if !ok {
		return DependencyKeyDoesntExist
	}
	dependencyData, ok := val.(DependencyData)
	if !ok {
		return DependencyAssertionFailure
	}
	newRemoteDependency := appinsights.NewRemoteDependencyTelemetry(dependencyData.Name, dependencyData.dependencyType, dependencyData.target, dependencyData.success)
	if requestID, ok := entry.Context.Value("request-id").(string); ok {
		newRemoteDependency.Id = requestID
	}
	hook.client.Track(newRemoteDependency)
	return nil
}

func (hook *AppInsightsHook) fireRequest(entry *logrus.Entry) error {
	val, ok := entry.Data[RequestKey]
	if !ok {
		return RequestKeyDoesntExist
	}
	requestData, ok := val.(RequestData)
	if !ok {
		return RequestAssertionFailure
	}
	requestDependency := appinsights.NewRequestTelemetry(requestData.Method, requestData.Url, requestData.Duration, requestData.responseCode)
	if requestID, ok := entry.Context.Value("request-id").(string); ok {
		requestDependency.Id = requestID
	}
	hook.client.Track(requestDependency)
	return nil
}

func (hook *AppInsightsHook) buildTrace(entry *logrus.Entry) (*appinsights.TraceTelemetry, error) {
	// Add the message as a field if it isn't already
	if _, ok := entry.Data["message"]; !ok {
		entry.Data["message"] = entry.Message
	}

	level := levelMap[entry.Level]
	trace := appinsights.NewTraceTelemetry(entry.Message, level)
	if trace == nil {
		return nil, fmt.Errorf("Could not create telemetry trace with entry %+v", entry)
	}
	for k, v := range entry.Data {
		if _, ok := hook.ignoreFields[k]; ok {
			continue
		}
		if fn, ok := hook.filters[k]; ok {
			v = fn(v) // apply custom filter
		} else {
			v = formatData(v) // use default formatter
		}
		trace.Properties[k] = fmt.Sprintf("%v", v)
	}
	//trace.Properties["source_level"] = entry.Level.String()
	//trace.Properties["source_timestamp"] = entry.Time.String()
	return trace, nil
}

// formatData returns value as a suitable format.
func formatData(value interface{}) (formatted interface{}) {
	switch value := value.(type) {
	case json.Marshaler:
		return value
	case error:
		return value.Error()
	case fmt.Stringer:
		return value.String()
	default:
		return value
	}
}

func stringPtr(str string) *string {
	return &str
}

func (hook *AppInsightsHook) _appInsightException(msg interface{}, skip int, requestID string) {
	switch msg.(type) {
	case error, string, fmt.Stringer: // currently these are the only supported types in RequestError
	default:
		msg = fmt.Errorf("%v", msg)
	}

	properties := make(map[string]string)
	properties["request-id"] = requestID
	// defined explicitly due to https://github.com/microsoft/ApplicationInsights-Go/issues/47
	hook.client.Track(&appinsights.ExceptionTelemetry{
		Error:         msg,
		Frames:        appinsights.GetCallstack(3 + skip),
		SeverityLevel: appinsights.Error,
		BaseTelemetry: appinsights.BaseTelemetry{
			Timestamp:  clock.NewClock().Now(),
			Tags:       make(contracts.ContextTags),
			Properties: properties,
		},
		BaseTelemetryMeasurements: appinsights.BaseTelemetryMeasurements{
			Measurements: make(map[string]float64),
		},
	})
}
