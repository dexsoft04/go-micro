// Package fly provides Fly.io specific client selector utilities for Go Micro
package fly

import (
	"os"

	"go-micro.dev/v5/registry"
	"go-micro.dev/v5/selector"
)

// SelectByMachineID creates a filter to select services running on a specific Fly.io machine
func SelectByMachineID(machineID string) selector.SelectOption {
	return selector.WithFilter(func(services []*registry.Service) []*registry.Service {
		filtered := make([]*registry.Service, 0)
		
		for _, service := range services {
			filteredNodes := make([]*registry.Node, 0)
			for _, node := range service.Nodes {
				if nodeMachineID, ok := node.Metadata["machine_id"]; ok && nodeMachineID == machineID {
					filteredNodes = append(filteredNodes, node)
				}
			}
			
			if len(filteredNodes) > 0 {
				s := *service
				s.Nodes = filteredNodes
				filtered = append(filtered, &s)
			}
		}
		
		return filtered
	})
}

// SelectByRegion creates a filter to select services running in a specific Fly.io region
func SelectByRegion(region string) selector.SelectOption {
	return selector.WithFilter(func(services []*registry.Service) []*registry.Service {
		filtered := make([]*registry.Service, 0)
		
		for _, service := range services {
			filteredNodes := make([]*registry.Node, 0)
			for _, node := range service.Nodes {
				if nodeRegion, ok := node.Metadata["region"]; ok && nodeRegion == region {
					filteredNodes = append(filteredNodes, node)
				}
			}
			
			if len(filteredNodes) > 0 {
				s := *service
				s.Nodes = filteredNodes
				filtered = append(filtered, &s)
			}
		}
		
		return filtered
	})
}

// SelectByAppName creates a filter to select services from a specific Fly.io app
func SelectByAppName(appName string) selector.SelectOption {
	return selector.WithFilter(func(services []*registry.Service) []*registry.Service {
		filtered := make([]*registry.Service, 0)
		
		for _, service := range services {
			filteredNodes := make([]*registry.Node, 0)
			for _, node := range service.Nodes {
				if nodeAppName, ok := node.Metadata["app_name"]; ok && nodeAppName == appName {
					filteredNodes = append(filteredNodes, node)
				}
			}
			
			if len(filteredNodes) > 0 {
				s := *service
				s.Nodes = filteredNodes
				filtered = append(filtered, &s)
			}
		}
		
		return filtered
	})
}

// SelectByPlatform creates a filter to select services running on Fly.io platform
func SelectByPlatform() selector.SelectOption {
	return selector.WithFilter(func(services []*registry.Service) []*registry.Service {
		filtered := make([]*registry.Service, 0)
		
		for _, service := range services {
			filteredNodes := make([]*registry.Node, 0)
			for _, node := range service.Nodes {
				if platform, ok := node.Metadata["platform"]; ok && platform == "fly.io" {
					filteredNodes = append(filteredNodes, node)
				}
			}
			
			if len(filteredNodes) > 0 {
				s := *service
				s.Nodes = filteredNodes
				filtered = append(filtered, &s)
			}
		}
		
		return filtered
	})
}

// SelectNearest creates a filter to prefer services in the same region as current machine
// Falls back to any region if no services found in current region
func SelectNearest() selector.SelectOption {
	currentRegion := getCurrentRegion()
	
	return selector.WithFilter(func(services []*registry.Service) []*registry.Service {
		if currentRegion == "" {
			return services // No region info available, return all
		}
		
		// First try to find services in the same region
		sameRegionServices := make([]*registry.Service, 0)
		
		for _, service := range services {
			sameRegionNodes := make([]*registry.Node, 0)
			for _, node := range service.Nodes {
				if nodeRegion, ok := node.Metadata["region"]; ok && nodeRegion == currentRegion {
					sameRegionNodes = append(sameRegionNodes, node)
				}
			}
			
			if len(sameRegionNodes) > 0 {
				s := *service
				s.Nodes = sameRegionNodes
				sameRegionServices = append(sameRegionServices, &s)
			}
		}
		
		// Return same region services if found, otherwise return all
		if len(sameRegionServices) > 0 {
			return sameRegionServices
		}
		
		return services
	})
}

// SelectByMultipleCriteria creates a filter that combines multiple criteria
func SelectByMultipleCriteria(machineID, region, appName string) selector.SelectOption {
	return selector.WithFilter(func(services []*registry.Service) []*registry.Service {
		filtered := make([]*registry.Service, 0)
		
		for _, service := range services {
			filteredNodes := make([]*registry.Node, 0)
			for _, node := range service.Nodes {
				match := true
				
				// Check machine ID if specified
				if machineID != "" {
					if nodeMachineID, ok := node.Metadata["machine_id"]; !ok || nodeMachineID != machineID {
						match = false
					}
				}
				
				// Check region if specified
				if region != "" && match {
					if nodeRegion, ok := node.Metadata["region"]; !ok || nodeRegion != region {
						match = false
					}
				}
				
				// Check app name if specified
				if appName != "" && match {
					if nodeAppName, ok := node.Metadata["app_name"]; !ok || nodeAppName != appName {
						match = false
					}
				}
				
				if match {
					filteredNodes = append(filteredNodes, node)
				}
			}
			
			if len(filteredNodes) > 0 {
				s := *service
				s.Nodes = filteredNodes
				filtered = append(filtered, &s)
			}
		}
		
		return filtered
	})
}

// getCurrentRegion gets the current Fly.io region from environment
func getCurrentRegion() string {
	return os.Getenv("FLY_REGION")
}