package integration

import (
	uuid "github.com/satori/go.uuid"
)

type (
	Integration struct {
		Version  string              `json:"version"`
		Kind     string              `json:"kind"`
		Metadata IntegrationMetadata `json:"metadata"`
		Spec     IntegrationSpec     `json:"spec"`
	}

	IntegrationMetadata struct {
		Name   string            `json:"name"`
		ID     string            `json:"id"`
		Labels map[string]string `json:"labels"`
	}

	IntegrationSpec struct {
		URL      string              `json:"url"`
		Secret   string              `json:"secret"`
		Triggers []string            `json:"triggers"`
		Resource IntegrationResource `json:"resource"`
	}

	IntegrationResource struct {
		Chart string `json:"chart"`
		Repo  string `json:"repo"`
	}

	IntegrationOptions struct {
		Name     string            `json:"name"`
		Labels   map[string]string `json:"labels"`
		URL      string            `json:"url"`
		Secret   string            `json:"secret"`
		Chart    string            `json:"chart"`
		Repo     string            `json:"repo"`
		Triggers []string          `json:"triggers"`
	}
)

func NewIntegration(spec IntegrationSpec, meta IntegrationMetadata) Integration {
	return Integration{
		Kind:     integrationKind,
		Version:  integrationVersion,
		Metadata: meta,
		Spec:     spec,
	}
}

func NewIntegrationFromOptions(opt *IntegrationOptions) Integration {
	meta := IntegrationMetadata{
		Name:   opt.Name,
		ID:     uuid.NewV4().String(),
		Labels: opt.Labels,
	}
	resource := IntegrationResource{
		Repo:  opt.Repo,
		Chart: opt.Chart,
	}
	spec := IntegrationSpec{
		Triggers: opt.Triggers,
		Resource: resource,
		URL:      opt.URL,
		Secret:   opt.Secret,
	}
	return NewIntegration(spec, meta)
}
