package processor

import (
	"bytes"
	"context"
	"encoding/json"
	cdsv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	edsv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	ldsv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	rdsv3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	accessloggerstreamv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/access_loggers/stream/v3"
	ratelimitv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ratelimit/v3"
	routerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	connectionmanagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	rlsconfigv3 "github.com/envoyproxy/go-control-plane/ratelimit/config/ratelimit/v3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	"gopkg.in/yaml.v3"
	"os"
	"strconv"
	"time"
	"xDS/internal/observer"
	"xDS/internal/xdscache"
)

var unmarshalOptions = protojson.UnmarshalOptions{
	DiscardUnknown: false,
	Resolver:       &TypeResolver{},
}

type Processor struct {
	cache    cache.SnapshotCache
	nodeID   string
	xdsCache xdscache.XDSCache
}

func NewProcessor(cache cache.SnapshotCache, nodeID string) *Processor {
	return &Processor{
		cache:  cache,
		nodeID: nodeID,
		xdsCache: xdscache.XDSCache{
			Data: make(map[string][]types.Resource),
		},
	}
}

// ProcessFile takes a file and generates an xDS snapshot
func (p *Processor) ProcessFile(msg observer.NotifyMessage) {
	if msg.IsNotSupported() {
		return
	}
	log.Infof("Processing file: %s, operation: %v", msg.FilePath, msg.OperationName())

	if msg.Operation == observer.Remove {
		p.xdsCache.RemoveConfig(msg.FilePath)
	} else {
		cfg, err := loadCfg(msg)
		if err != nil {
			log.Errorf("Load msg %v error: %v", msg, err)
			return
		}
		p.xdsCache.SetConfig(msg.FilePath, cfg)
	}

	resources := map[resource.Type][]types.Resource{
		resource.ClusterType:         p.xdsCache.Cds(),
		resource.ListenerType:        p.xdsCache.Lds(),
		resource.RouteType:           p.xdsCache.Rds(),
		resource.RateLimitConfigType: p.xdsCache.Rls(),
	}
	version := strconv.FormatInt(time.Now().Unix(), 10)
	snapshot, _ := cache.NewSnapshot(version, resources)
	if err := snapshot.Consistent(); err != nil {
		log.Errorf("snapshot inconsistency: %+v\n\n\n%+v", snapshot, err)
		return
	}
	s, _ := json.Marshal(snapshot)
	log.Debugf("will serve snapshot %+v", string(s))

	err := p.cache.SetSnapshot(context.Background(), p.nodeID, snapshot)
	if err != nil {
		log.Errorf("UpdateSnapShot error: %v", err)
		return
	}

}

func loadCfg(msg observer.NotifyMessage) ([]types.Resource, error) {
	configYaml, err := os.ReadFile(msg.FilePath)
	if err != nil {
		return nil, err
	}

	configs := bytes.Split(configYaml, []byte("---"))
	var xds []types.Resource

	for _, config := range configs {
		config = bytes.TrimSpace(config)
		if len(config) == 0 {
			continue
		}

		cfg := correspondResource(msg)
		if err := configToResource(config, cfg); err != nil {
			return nil, err
		}
		xds = append(xds, cfg)
	}
	return xds, nil
}

func correspondResource(file observer.NotifyMessage) types.Resource {
	if file.IsLds() {
		return &ldsv3.Listener{}
	} else if file.IsRds() {
		return &rdsv3.Route{}
	} else if file.IsEds() {
		return &edsv3.Endpoint{}
	} else if file.IsCds() {
		return &cdsv3.Cluster{}
	} else if file.IsRls() {
		return &rlsconfigv3.RateLimitConfig{}
	}
	return nil
}

func configToResource(config []byte, r types.Resource) error {
	var jsonData map[string]interface{}
	err := yaml.Unmarshal(config, &jsonData)
	if err != nil {
		return err
	}
	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		return err
	}
	if err := unmarshalOptions.Unmarshal(jsonBytes, r); err != nil {
		return err
	}
	return nil
}

// TypeResolver implements protoregistry.ExtensionTypeResolver and protoregistry.MessageTypeResolver to resolve google.protobuf.Any types
type TypeResolver struct{}

func (r *TypeResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}

// FindMessageByURL links the message type url to the specific message type
// TODO: If there's other message type can be passed in google.protobuf.Any, the typeUrl and messageType need to be added to this method to make sure it can be parsed and output correctly.
func (r *TypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	switch url {
	case "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager":
		connectionManager := connectionmanagerv3.HttpConnectionManager{}
		return connectionManager.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router":
		router := routerv3.Router{}
		return router.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.extensions.access_loggers.stream.v3.StdoutAccessLog":
		stdoutAccessLog := accessloggerstreamv3.StdoutAccessLog{}
		return stdoutAccessLog.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit":
		rateLimit := ratelimitv3.RateLimit{}
		return rateLimit.ProtoReflect().Type(), nil
	default:
		dummy := anypb.Any{}
		return dummy.ProtoReflect().Type(), nil
	}
}

func (r *TypeResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

func (r *TypeResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}
