package xdscache

import (
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"strings"
	"xDS/internal/constant"
)

type XDSCache struct {
	Data map[string][]types.Resource
}

func (xds *XDSCache) SetConfig(name string, config []types.Resource) *XDSCache {
	xds.Data[name] = config
	return xds
}

func (xds *XDSCache) RemoveConfig(name string) *XDSCache {
	delete(xds.Data, name)
	return xds
}

// Cds Cluster dynamic source
func (xds *XDSCache) Cds() []types.Resource {
	return xds.resources(constant.ClusterFileSuffix)
}

// Rds Route dynamic source
func (xds *XDSCache) Rds() []types.Resource {
	return xds.resources(constant.RouteFileSuffix)
}

// Lds Listener dynamic source
func (xds *XDSCache) Lds() []types.Resource {
	return xds.resources(constant.ListenerFileSuffix)
}

// Eds Endpoints dynamic source
func (xds *XDSCache) Eds() []types.Resource {
	return xds.resources(constant.EndpointFileSuffix)
}

// Rls RateLimitDescriptors
func (xds *XDSCache) Rls() []types.Resource {
	return xds.resources(constant.RatelimitFileSuffix)
}

func (xds *XDSCache) resources(typeSuffix string) []types.Resource {
	var resources []types.Resource
	for name, c := range xds.Data {
		if strings.HasSuffix(name, typeSuffix) {
			resources = append(resources, c...)
		}
	}
	return resources
}
