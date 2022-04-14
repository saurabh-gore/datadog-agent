// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package inferredspan

import (
	"strings"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config"
	rand "github.com/DataDog/datadog-agent/pkg/serverless/random"
	"github.com/DataDog/datadog-agent/pkg/serverless/tags"
	"github.com/DataDog/datadog-agent/pkg/trace/api"
	"github.com/DataDog/datadog-agent/pkg/trace/info"
	"github.com/DataDog/datadog-agent/pkg/trace/pb"
	"github.com/DataDog/datadog-agent/pkg/trace/sampler"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const (
	// tagInferredSpanTagSource is the key to the meta tag
	// that lets us know whether this span should inherit its tags.
	// Expected options are "lambda" and "self"
	tagInferredSpanTagSource = "_inferred_span.tag_source"

	// additional function specific tag keys to ignore
	functionVersionTagKey = "function_version"
	coldStartTagKey       = "cold_start"
)

// InferredSpan contains the pb.Span and Async information
// of the inferredSpan for the current invocation
type InferredSpan struct {
	Span    *pb.Span
	IsAsync bool
	// CurrentInvocationStartTime is the start time of the
	// current invocation not he inferred span. It is used
	// for async function calls to calculate the duration.
	CurrentInvocationStartTime time.Time
}

var functionTagsToIgnore = []string{
	tags.FunctionARNKey,
	tags.FunctionNameKey,
	tags.ExecutedVersionKey,
	tags.EnvKey,
	tags.VersionKey,
	tags.ServiceKey,
	tags.RuntimeKey,
	tags.MemorySizeKey,
	tags.ArchitectureKey,
	functionVersionTagKey,
	coldStartTagKey,
}

// CheckIsInferredSpan determines if a span belongs to a managed service or not
// _inferred_span.tag_source = "self" => managed service span
// _inferred_span.tag_source = "lambda" or missing => lambda related span
func CheckIsInferredSpan(span *pb.Span) bool {
	return strings.Compare(span.Meta[tagInferredSpanTagSource], "self") == 0
}

// FilterFunctionTags filters out DD tags & function specific tags
func FilterFunctionTags(input map[string]string) map[string]string {

	if input == nil {
		return nil
	}

	output := make(map[string]string)
	for k, v := range input {
		output[k] = v
	}

	// filter out DD_TAGS & DD_EXTRA_TAGS
	ddTags := config.GetConfiguredTags(false)
	for _, tag := range ddTags {
		tagParts := strings.SplitN(tag, ":", 2)
		if len(tagParts) != 2 {
			log.Warnf("Cannot split tag %s", tag)
			continue
		}
		tagKey := tagParts[0]
		delete(output, tagKey)
	}

	// filter out function specific tags
	for _, tagKey := range functionTagsToIgnore {
		delete(output, tagKey)
	}

	return output
}

// RouteInferredSpan decodes the event and routes it to the correct
// enrichment function for that event source
func RouteInferredSpan(event string, inferredSpan InferredSpan) {
	// Parse the event into the EventKey struct
	eventSource, attributes := ParseEventSource(event)
	switch eventSource {
	case APIGATEWAY:
		EnrichInferredSpanWithAPIGatewayRESTEvent(attributes, inferredSpan)
	case HTTPAPI:
		EnrichInferredSpanWithAPIGatewayHTTPEvent(attributes, inferredSpan)
	case WEBSOCKET:
		EnrichInferredSpanWithAPIGatewayWebsocketEvent(attributes, inferredSpan)
	}
}

// CompleteInferredSpan finishes the inferred span and passes it
// as an API payload to be processed by the trace agent
func CompleteInferredSpan(
	processTrace func(p *api.Payload),
	endTime time.Time,
	isError bool,
	inferredSpan InferredSpan) {
	log.Debug("Inside CompleteInferredSpan")
	if inferredSpan.IsAsync {
		log.Debug("This is async")
		inferredSpan.Span.Duration = inferredSpan.CurrentInvocationStartTime.Unix() - inferredSpan.Span.Start
	} else {
		log.Debug("This is not async")
		inferredSpan.Span.Duration = endTime.Unix() - inferredSpan.Span.Start
	}
	if isError {
		inferredSpan.Span.Error = 1
	}
	traceChunk := &pb.TraceChunk{
		Priority: int32(sampler.PriorityNone),
		Origin:   "lambda",
		Spans:    []*pb.Span{inferredSpan.Span},
	}

	tracerPayload := &pb.TracerPayload{
		Chunks: []*pb.TraceChunk{traceChunk},
	}

	processTrace(&api.Payload{
		Source:        info.NewReceiverStats().GetTagStats(info.Tags{}),
		TracerPayload: tracerPayload,
	})
}

// GenerateInferredSpan declares and initializes a new inferred span
// with the SpanID and TraceID
func GenerateInferredSpan(startTime time.Time) InferredSpan {
	var inferredSpan InferredSpan
	inferredSpan.Span = &pb.Span{}
	inferredSpan.Span.SpanID = rand.Random.Uint64()
	inferredSpan.Span.TraceID = rand.Random.Uint64()
	inferredSpan.CurrentInvocationStartTime = startTime
	log.Debug("Generated new Inferred span ", inferredSpan)
	return inferredSpan
}
