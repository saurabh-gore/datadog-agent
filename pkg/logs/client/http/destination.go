package http

import (
	"bytes"
	"context"
	"errors"
	"expvar"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/logs/client"
	"github.com/DataDog/datadog-agent/pkg/logs/config"
	"github.com/DataDog/datadog-agent/pkg/logs/message"
	"github.com/DataDog/datadog-agent/pkg/logs/metrics"
	"github.com/DataDog/datadog-agent/pkg/telemetry"
	"github.com/DataDog/datadog-agent/pkg/util/backoff"
	httputils "github.com/DataDog/datadog-agent/pkg/util/http"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/version"
)

// ContentType options,
const (
	TextContentType = "text/plain"
	JSONContentType = "application/json"
)

// HTTP errors.
var (
	errClient = errors.New("client error")
	errServer = errors.New("server error")
	tlmSend   = telemetry.NewCounter("logs_client_http_destination", "send", []string{"endpoint_host", "error"}, "Payloads sent")

	destinationExpVars = expvar.NewMap("http_destination")
	expVarIdleMsMapKey = "idleMs"
	expVarInUseMapKey  = "inUseMs"
)

// emptyPayload is an empty payload used to check HTTP connectivity without sending logs.
var emptyPayload = message.Payload{}

// Destination sends a payload over HTTP.
type Destination struct {
	sync.Mutex
	url                 string
	apiKey              string
	contentType         string
	host                string
	client              *httputils.ResetClient
	destinationsContext *client.DestinationsContext
	climit              chan struct{} // semaphore for limiting concurrent background sends
	backoff             backoff.Policy
	nbErrors            int
	blockedUntil        time.Time
	protocol            config.IntakeProtocol
	origin              config.IntakeOrigin
	lastError           error
	shouldRetry         bool
	expVars             *expvar.Map
}

// NewDestination returns a new Destination.
// If `maxConcurrentBackgroundSends` > 0, then at most that many background payloads will be sent concurrently, else
// there is no concurrency and the background sending pipeline will block while sending each payload.
// TODO: add support for SOCKS5
func NewDestination(endpoint config.Endpoint, contentType string, destinationsContext *client.DestinationsContext, maxConcurrentBackgroundSends int, shouldRetry bool, pipelineID int) *Destination {
	return newDestination(endpoint, contentType, destinationsContext, time.Second*10, maxConcurrentBackgroundSends, shouldRetry, pipelineID)
}

func newDestination(endpoint config.Endpoint, contentType string, destinationsContext *client.DestinationsContext, timeout time.Duration, maxConcurrentBackgroundSends int, shouldRetry bool, pipelineID int) *Destination {
	if maxConcurrentBackgroundSends < 0 {
		maxConcurrentBackgroundSends = 0
	}

	expVars := &expvar.Map{}
	expVars.AddFloat(expVarIdleMsMapKey, 0)
	expVars.AddFloat(expVarInUseMapKey, 0)
	destinationExpVars.Set(fmt.Sprintf("%s_%d", endpoint.Host, pipelineID), expVars)

	policy := backoff.NewPolicy(
		endpoint.BackoffFactor,
		endpoint.BackoffBase,
		endpoint.BackoffMax,
		endpoint.RecoveryInterval,
		endpoint.RecoveryReset,
	)

	return &Destination{
		host:                endpoint.Host,
		url:                 buildURL(endpoint),
		apiKey:              endpoint.APIKey,
		contentType:         contentType,
		client:              httputils.NewResetClient(endpoint.ConnectionResetInterval, httpClientFactory(timeout)),
		destinationsContext: destinationsContext,
		climit:              make(chan struct{}, maxConcurrentBackgroundSends),
		backoff:             policy,
		protocol:            endpoint.Protocol,
		origin:              endpoint.Origin,
		lastError:           nil,
		shouldRetry:         shouldRetry,
		expVars:             expVars,
	}
}

func errorToTag(err error) string {
	if err == nil {
		return "none"
	} else if _, ok := err.(*client.RetryableError); ok {
		return "retryable"
	} else {
		return "non-retryable"
	}
}

// Start starts reading the input channel
func (d *Destination) Start(input chan *message.Payload, isRetrying chan bool, output chan *message.Payload) {
	go func() {

		var startIdle = time.Now()

		for p := range input {
			d.expVars.AddFloat(expVarIdleMsMapKey, float64(time.Since(startIdle)/time.Millisecond))
			var startInUse = time.Now()

			d.sendConcurrent(p, isRetrying, output)

			d.expVars.AddFloat(expVarInUseMapKey, float64(time.Since(startInUse)/time.Millisecond))
			startIdle = time.Now()
		}
	}()
}

func (d *Destination) sendConcurrent(payload *message.Payload, isRetrying chan bool, output chan *message.Payload) {
	// if the channel is non-buffered then there is no concurrency and we block on sending each payload
	if cap(d.climit) == 0 {
		d.sendAndRetry(payload, isRetrying, output)
		return
	}

	go func() {
		d.climit <- struct{}{}
		go func() {
			d.sendAndRetry(payload, isRetrying, output)
			<-d.climit
		}()
	}()
}

// Send sends a payload over HTTP,
func (d *Destination) sendAndRetry(payload *message.Payload, isRetrying chan bool, output chan *message.Payload) {
	for {
		d.blockedUntil = time.Now().Add(d.backoff.GetBackoffDuration(d.nbErrors))
		if d.blockedUntil.After(time.Now()) {
			log.Debugf("%s: sleeping until %v before retrying", d.url, d.blockedUntil)
			d.waitForBackoff()
		}

		metrics.LogsSent.Add(int64(len(payload.Messages)))
		metrics.TlmLogsSent.Add(float64(len(payload.Messages)))
		err := d.unconditionalSend(payload)

		if err == context.Canceled {
			log.Warnf("Could not send payload: %v", err)
			return
		}

		if d.shouldRetry {
			d.Lock()
			if _, ok := err.(*client.RetryableError); ok {
				d.nbErrors = d.backoff.IncError(d.nbErrors)

				if d.lastError == nil && isRetrying != nil {
					isRetrying <- true
				}

				d.lastError = err
				d.Unlock()
				continue

			} else {
				d.nbErrors = d.backoff.DecError(d.nbErrors)

				if d.lastError != nil && isRetrying != nil {
					isRetrying <- false
				}

				d.lastError = nil
				d.Unlock()
			}
		}

		output <- payload
		return
	}
}

func (d *Destination) unconditionalSend(payload *message.Payload) (err error) {
	defer func() {
		tlmSend.Inc(d.host, errorToTag(err))
	}()

	ctx := d.destinationsContext.Context()

	if err != nil {
		return err
	}
	metrics.BytesSent.Add(int64(len(payload.Encoded)))
	metrics.EncodedBytesSent.Add(int64(len(payload.Encoded)))

	req, err := http.NewRequest("POST", d.url, bytes.NewReader(payload.Encoded))
	if err != nil {
		// the request could not be built,
		// this can happen when the method or the url are valid.
		return err
	}
	req.Header.Set("DD-API-KEY", d.apiKey)
	req.Header.Set("Content-Type", d.contentType)
	req.Header.Set("Content-Encoding", payload.Encoding)
	if d.protocol != "" {
		req.Header.Set("DD-PROTOCOL", string(d.protocol))
	}
	if d.origin != "" {
		req.Header.Set("DD-EVP-ORIGIN", string(d.origin))
		req.Header.Set("DD-EVP-ORIGIN-VERSION", version.AgentVersion)
	}
	req = req.WithContext(ctx)

	then := time.Now()
	resp, err := d.client.Do(req)

	latency := time.Since(then).Milliseconds()
	metrics.TlmSenderLatency.Observe(float64(latency))
	metrics.SenderLatency.Set(latency)

	if err != nil {
		metrics.DestinationErrors.Add(1)
		metrics.TlmDestinationErrors.Inc()

		if ctx.Err() == context.Canceled {
			return ctx.Err()
		}
		// most likely a network or a connect error, the callee should retry.
		return client.NewRetryableError(err)
	}

	defer resp.Body.Close()
	response, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// the read failed because the server closed or terminated the connection
		// *after* serving the request.
		return err
	}
	if resp.StatusCode >= 400 {
		log.Warnf("failed to post http payload. code=%d host=%s response=%s", resp.StatusCode, d.host, string(response))
	}
	if resp.StatusCode == 429 || resp.StatusCode >= 500 {
		// the server could not serve the request, most likely because of an
		// internal error or, (429) because it is overwhelmed
		return client.NewRetryableError(errServer)
	} else if resp.StatusCode >= 400 {
		// the logs-agent is likely to be misconfigured,
		// the URL or the API key may be wrong.
		return errClient
	} else {
		return nil
	}
}

func httpClientFactory(timeout time.Duration) func() *http.Client {
	return func() *http.Client {
		return &http.Client{
			Timeout: timeout,
			// reusing core agent HTTP transport to benefit from proxy settings.
			Transport: httputils.CreateHTTPTransport(),
		}
	}
}

// buildURL buils a url from a config endpoint.
func buildURL(endpoint config.Endpoint) string {
	var scheme string
	if endpoint.UseSSL {
		scheme = "https"
	} else {
		scheme = "http"
	}
	var address string
	if endpoint.Port != 0 {
		address = fmt.Sprintf("%v:%v", endpoint.Host, endpoint.Port)
	} else {
		address = endpoint.Host
	}
	url := url.URL{
		Scheme: scheme,
		Host:   address,
	}
	if endpoint.Version == config.EPIntakeVersion2 && endpoint.TrackType != "" {
		url.Path = fmt.Sprintf("/api/v2/%s", endpoint.TrackType)
	} else {
		url.Path = "/v1/input"
	}
	return url.String()
}

// CheckConnectivity check if sending logs through HTTP works
func CheckConnectivity(endpoint config.Endpoint) config.HTTPConnectivity {
	log.Info("Checking HTTP connectivity...")
	ctx := client.NewDestinationsContext()
	ctx.Start()
	defer ctx.Stop()
	// Lower the timeout to 5s because HTTP connectivity test is done synchronously during the agent bootstrap sequence
	destination := newDestination(endpoint, JSONContentType, ctx, time.Second*5, 0, false, 0)
	log.Infof("Sending HTTP connectivity request to %s...", destination.url)
	err := destination.unconditionalSend(&emptyPayload)
	if err != nil {
		log.Warnf("HTTP connectivity failure: %v", err)
	} else {
		log.Info("HTTP connectivity successful")
	}
	return err == nil
}

func (d *Destination) waitForBackoff() {
	ctx, cancel := context.WithDeadline(d.destinationsContext.Context(), d.blockedUntil)
	defer cancel()
	<-ctx.Done()
}
