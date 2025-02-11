// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build kubeapiserver
// +build kubeapiserver

package apiserver

import (
	"regexp"

	klog "k8s.io/klog"
)

var supressedWarning = regexp.MustCompile(`.*is deprecated in v.*`)

type CustomWarningLogger struct{}

// HandleWarningHeader suppresses some warning logs
// TODO: Remove custom warning logger when we remove usage of ComponentStatus
func (CustomWarningLogger) HandleWarningHeader(code int, agent string, message string) {
	if code != 299 || len(message) == 0 || supressedWarning.MatchString(message) {
		return
	}

	klog.Warning(message)
}
