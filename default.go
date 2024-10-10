package micro

import (
	"github.com/philchia/agollo/v4"
	"github.com/zigo2048/mcbeam-common-lib/common/config"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/debug"
	"github.com/zigo2048/mcbeam-common-lib/common/wrapper/wrapper"
	"github.com/zigo2048/mcbeam-common-lib/plugins/config/apollo/v3"
	"go-micro.dev/v5/logger"
	"go-micro.dev/v5/server"
	"os"
	"path/filepath"

	_ "github.com/micro/plugins/v5/broker/nats"
	_ "github.com/micro/plugins/v5/registry/etcd"
)

func initDefaultConfig() {
	config.DefaultConfig = apollo.NewConfig(apollo.WithConfig(&agollo.Conf{
		AppID:          os.Getenv("MICRO_NAMESPACE"),
		Cluster:        "default",
		NameSpaceNames: []string{os.Getenv("MICRO_SERVICE_NAME") + ".yaml"},
		MetaAddr:       os.Getenv("MICRO_CONFIG_ADDRESS"),
		CacheDir:       filepath.Join(os.TempDir(), "apollo"),
	}))

	var err error
	err = server.DefaultServer.Init(
		server.WrapHandler(wrapper.AuthHandler()),
		server.WrapHandler(debug.WrapperHandler),
	)
	if nil != err {
		logger.Fatalf("init default server err:%s", err)
	}
}
