package micro

import (
	_ "github.com/micro/plugins/v5/broker/nats"
	_ "github.com/micro/plugins/v5/registry/etcd"
	"github.com/micro/plugins/v5/wrapper/trace/opentelemetry"
	"github.com/philchia/agollo/v4"
	"github.com/zigo2048/mcbeam-common-lib/common/config"
	"github.com/zigo2048/mcbeam-common-lib/plugins/config/apollo/v3"
	"go-micro.dev/v5/client"
	_ "go-micro.dev/v5/transport/grpc"
	"os"
	"path/filepath"
)

func init() {
	client.DefaultClient = client.NewClient(client.Wrap(opentelemetry.NewClientWrapper()))
}

func initDefaultConfig() {
	config.DefaultConfig = apollo.NewConfig(apollo.WithConfig(&agollo.Conf{
		AppID:          os.Getenv("MICRO_NAMESPACE"),
		Cluster:        "default",
		NameSpaceNames: []string{os.Getenv("MICRO_SERVICE_NAME") + ".yaml"},
		MetaAddr:       os.Getenv("MICRO_CONFIG_ADDRESS"),
		CacheDir:       filepath.Join(os.TempDir(), "apollo"),
	}))
}
