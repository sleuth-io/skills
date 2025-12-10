package clients

import (
	"context"
	"sync"
)

// Orchestrator coordinates installation across multiple clients
type Orchestrator struct {
	registry *Registry
}

// NewOrchestrator creates a new installation orchestrator
func NewOrchestrator(registry *Registry) *Orchestrator {
	return &Orchestrator{registry: registry}
}

// InstallToAll installs artifacts to all detected clients concurrently
func (o *Orchestrator) InstallToAll(ctx context.Context,
	artifacts []*ArtifactBundle,
	scope *InstallScope,
	options InstallOptions) map[string]InstallResponse {
	clients := o.registry.DetectInstalled()
	return o.InstallToClients(ctx, artifacts, scope, options, clients)
}

// InstallToClients installs artifacts to specific clients concurrently
func (o *Orchestrator) InstallToClients(ctx context.Context,
	artifacts []*ArtifactBundle,
	scope *InstallScope,
	options InstallOptions,
	targetClients []Client) map[string]InstallResponse {

	// Install to clients concurrently
	results := make(map[string]InstallResponse)
	resultsMu := sync.Mutex{}
	wg := sync.WaitGroup{}

	for _, client := range targetClients {
		wg.Add(1)
		go func(client Client) {
			defer wg.Done()

			// Filter artifacts by client compatibility and scope support
			compatibleArtifacts := o.filterArtifacts(artifacts, client, scope)

			if len(compatibleArtifacts) == 0 {
				resultsMu.Lock()
				results[client.ID()] = InstallResponse{
					Results: []ArtifactResult{
						{
							Status:  StatusSkipped,
							Message: "No compatible artifacts",
						},
					},
				}
				resultsMu.Unlock()
				return
			}

			// Let the client handle installation however it wants
			req := InstallRequest{
				Artifacts: compatibleArtifacts,
				Scope:     scope,
				Options:   options,
			}

			resp, err := client.InstallArtifacts(ctx, req)
			if err != nil {
				// Client returned error - ensure all results marked as failed
				for i := range resp.Results {
					if resp.Results[i].Status != StatusFailed {
						resp.Results[i].Status = StatusFailed
						if resp.Results[i].Error == nil {
							resp.Results[i].Error = err
						}
					}
				}
			}

			resultsMu.Lock()
			results[client.ID()] = resp
			resultsMu.Unlock()
		}(client)
	}

	wg.Wait()
	return results
}

// filterArtifacts returns artifacts compatible with client and scope
func (o *Orchestrator) filterArtifacts(artifacts []*ArtifactBundle,
	client Client,
	scope *InstallScope) []*ArtifactBundle {
	compatible := make([]*ArtifactBundle, 0)

	for _, bundle := range artifacts {
		// Check if client supports this artifact type
		if !client.SupportsArtifactType(bundle.Artifact.Type) {
			continue
		}

		// If artifact is scoped to repo/path and this is a global scope,
		// skip it (client doesn't support repo scope)
		if !bundle.Artifact.IsGlobal() && scope.Type == ScopeGlobal {
			continue
		}

		compatible = append(compatible, bundle)
	}

	return compatible
}

// HasAnyErrors checks if any client installation failed
func HasAnyErrors(results map[string]InstallResponse) bool {
	for _, resp := range results {
		for _, result := range resp.Results {
			if result.Status == StatusFailed {
				return true
			}
		}
	}
	return false
}
