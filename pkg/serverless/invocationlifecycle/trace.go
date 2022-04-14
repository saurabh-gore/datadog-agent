// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package invocationlifecycle

import (
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"time"

	rand "github.com/DataDog/datadog-agent/pkg/serverless/random"
	"github.com/DataDog/datadog-agent/pkg/trace/api"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	functionNameEnvVar = "AWS_LAMBDA_FUNCTION_NAME"
)

// executionStartInfo is saved information from when an execution span was started
type executionStartInfo struct {
	startTime time.Time
	traceID   uint64
	spanID    uint64
	parentID  uint64
}
type invocationPayload struct {
	Headers map[string]string `json:"headers"`
}

// currentExecutionInfo represents information from the start of the current execution span
var currentExecutionInfo executionStartInfo

// startExecutionSpan records information from the start of the invocation.
// It should be called at the start of the invocation.
func startExecutionSpan(startTime time.Time, rawPayload string) {
	currentExecutionInfo.startTime = startTime
	currentExecutionInfo.parentID = 0

	payload := convertRawPayload(rawPayload)

	if InferredSpansEnabled {
		currentExecutionInfo.traceID = inferredSpan.Span.TraceID
		currentExecutionInfo.parentID = inferredSpan.Span.SpanID
	} else {
		currentExecutionInfo.traceID = rand.Random.Uint64()
		currentExecutionInfo.spanID = rand.Random.Uint64()
	}

	if payload.Headers != nil {
		traceID, e1 := strconv.ParseUint(payload.Headers[TraceIDHeader], 0, 64)
		parentID, e2 := strconv.ParseUint(payload.Headers[ParentIDHeader], 0, 64)

		if e1 == nil {
			currentExecutionInfo.traceID = traceID
			if InferredSpansEnabled {
				inferredSpan.Span.TraceID = traceID
			}
		}

		if e2 == nil {
			currentExecutionInfo.parentID = parentID
			if InferredSpansEnabled {
				inferredSpan.Span.ParentID = parentID
				currentExecutionInfo.parentID = inferredSpan.Span.SpanID
			}
		}
	}
}

// endExecutionSpan builds the function execution span and sends it to the intake.
// It should be called at the end of the invocation.
func endExecutionSpan(processTrace func(p *api.Payload), requestID string, endTime time.Time, isError bool) {
	duration := endTime.UnixNano() - currentExecutionInfo.startTime.UnixNano()

	executionSpan := &pb.Span{
		Service:  "aws.lambda", // will be replaced by the span processor
		Name:     "aws.lambda",
		Resource: os.Getenv(functionNameEnvVar),
		Type:     "serverless",
		TraceID:  currentExecutionInfo.traceID,
		SpanID:   currentExecutionInfo.spanID,
		ParentID: currentExecutionInfo.parentID,
		Start:    currentExecutionInfo.startTime.UnixNano(),
		Duration: duration,
		Meta: map[string]string{
			"request_id": requestID,
		},
	}

	if isError {
		executionSpan.Error = 1
	}

	traceChunk := &pb.TraceChunk{
		Priority: int32(sampler.PriorityNone),
		Spans:    []*pb.Span{executionSpan},
	}

	tracerPayload := &pb.TracerPayload{
		Chunks: []*pb.TraceChunk{traceChunk},
	}

	processTrace(&api.Payload{
		Source:        info.NewReceiverStats().GetTagStats(info.Tags{}),
		TracerPayload: tracerPayload,
	})
}

func convertRawPayload(rawPayload string) invocationPayload {
	//Need to remove unwanted text from the initial payload
	reg := regexp.MustCompile(`{(?:|(.*))*}`)
	subString := reg.FindString(rawPayload)

	payload := invocationPayload{}

	err := json.Unmarshal([]byte(subString), &payload)
	if err != nil {
		log.Debug("Could not unmarshal the invocation event payload")
	}

	return payload
}

// TraceID returns the current TraceID
func TraceID() uint64 {
	return currentExecutionInfo.traceID
}

// SpanID returns the current SpanID
func SpanID() uint64 {
	return currentExecutionInfo.spanID
}

// Used for setting the InferredSpansEnable var when testing
// This allows us to avoid setting env vars in tests
func SetVarForTest() {
	InferredSpansEnabled = true
}
