package cbi_plugin_v1

import (
	"strings"

	crd "github.com/containerbuilding/cbi/pkg/apis/cbi/v1alpha1"
)

var (
	// PredefinedLabelPrefixes is the set of predefined plugin label prefixes.
	// Although plugin labels are decoupled from k8s object labels,
	// we try to follow the k8s convention.
	//
	// At the moment, we don't use long prefix such as "plugin.cbi.containerbuilding.github.io".
	// However, the convention might change in v1 GA.
	PredefinedLabelPrefixes = []string{
		"plugin.",
		"language.",
		"context.",
	}
)

const (
	// LPluginName is the NON-UNIQUE name of the plugin.
	// LPluginName MUST be always present.
	//
	// Example values: "buildkit", "buildah", ...
	LPluginName = "plugin.name"
	// TODO: add LPluginVersion = "v1alpha1"?
)

func LLanguage(k crd.LanguageKind) string {
	return "language." + strings.ToLower(string(k))
}

func LContext(k crd.ContextKind) string {
	return "context." + strings.ToLower(string(k))
}
