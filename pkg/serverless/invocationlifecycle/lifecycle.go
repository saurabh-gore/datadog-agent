// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package invocationlifecycle

import (
	"os"
	"strconv"

	"github.com/DataDog/datadog-agent/pkg/aggregator"
	serverlessLog "github.com/DataDog/datadog-agent/pkg/serverless/logs"
	serverlessMetrics "github.com/DataDog/datadog-agent/pkg/serverless/metrics"
	"github.com/DataDog/datadog-agent/pkg/serverless/trace/inferredspan"
	"github.com/DataDog/datadog-agent/pkg/trace/api"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// LifecycleProcessor is a InvocationProcessor implementation
type LifecycleProcessor struct {
	ExtraTags           *serverlessLog.Tags
	ProcessTrace        func(p *api.Payload)
	Demux               aggregator.Demultiplexer
	DetectLambdaLibrary func() bool
}

var traceEnabled, _ = strconv.ParseBool(os.Getenv("DD_TRACE_ENABLED"))
var managedServiceEnabled, _ = strconv.ParseBool(os.Getenv("DD_TRACE_MANAGED_SERVICES"))

// InferredSpansEnabled tells us if the Env Vars are enabled
// for inferred spans to be created
var InferredSpansEnabled = traceEnabled && managedServiceEnabled
var inferredSpan inferredspan.InferredSpan

// OnInvokeStart is the hook triggered when an invocation has started
func (lp *LifecycleProcessor) OnInvokeStart(startDetails *InvocationStartDetails) {
	log.Debug("[lifecycle] onInvokeStart ------")
	log.Debug("[lifecycle] Invocation has started at :", startDetails.StartTime)
	log.Debug("[lifecycle] Invocation invokeEvent payload is :", startDetails.InvokeEventRawPayload)
	log.Debug("[lifecycle] ---------------------------------------")

	if !lp.DetectLambdaLibrary() {
		if InferredSpansEnabled {
			log.Debug("[lifecycle] Attempting to create inferred span")
			inferredSpan = inferredspan.GenerateInferredSpan(startDetails.StartTime)
			inferredspan.RouteInferredSpan(startDetails.InvokeEventRawPayload, inferredSpan)
		}

		startExecutionSpan(startDetails.StartTime, startDetails.InvokeEventRawPayload)
	}
}

// OnInvokeEnd is the hook triggered when an invocation has ended
func (lp *LifecycleProcessor) OnInvokeEnd(endDetails *InvocationEndDetails) {
	log.Debug("[lifecycle] onInvokeEnd ------")
	log.Debug("[lifecycle] Invocation has finished at :", endDetails.EndTime)
	log.Debug("[lifecycle] Invocation isError is :", endDetails.IsError)
	log.Debug("[lifecycle] ---------------------------------------")

	if !lp.DetectLambdaLibrary() {
		log.Debug("Creating and sending function execution span for invocation")
		endExecutionSpan(lp.ProcessTrace, endDetails.RequestID, endDetails.EndTime, endDetails.IsError)

		if InferredSpansEnabled {
			log.Debug("[lifecycle] Attempting to complete the inferred span")
			inferredspan.CompleteInferredSpan(
				lp.ProcessTrace, endDetails.EndTime, endDetails.IsError, inferredSpan,
			)
		}
	}

	if endDetails.IsError {
		serverlessMetrics.SendErrorsEnhancedMetric(
			lp.ExtraTags.Tags, endDetails.EndTime, lp.Demux,
		)
	}
}
