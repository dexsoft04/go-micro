package micro

import (
	"github.com/philchia/agollo/v4"
	"github.com/zigo2048/mcbeam-common-lib/common/config"
	"github.com/zigo2048/mcbeam-common-lib/plugins/config/apollo/v3"
	"os"
	"path/filepath"
)

func initDefaultConfig() {
	config.DefaultConfig = apollo.NewConfig(apollo.WithConfig(&agollo.Conf{
		AppID:          os.Getenv("MICRO_NAMESPACE"),
		Cluster:        "default",
		NameSpaceNames: []string{os.Getenv("MICRO_SERVICE_NAME") + ".yaml"},
		MetaAddr:       os.Getenv("MICRO_CONFIG_ADDRESS"),
		CacheDir:       filepath.Join(os.TempDir(), "apollo"),
	}))
}
