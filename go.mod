module github.com/Axway/agent-sdk

go 1.17

require (
	github.com/elastic/beats/v7 v7.7.1
	github.com/emersion/go-sasl v0.0.0-20200509203442-7bfe0ed36a21
	github.com/emersion/go-smtp v0.13.0
	github.com/emicklei/proto v1.9.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gabriel-vasile/mimetype v1.1.2
	github.com/getkin/kin-openapi v0.76.0
	github.com/golang-jwt/jwt v3.2.1+incompatible
	github.com/google/uuid v1.1.2
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563
	github.com/sirupsen/logrus v1.6.0
	github.com/snowzach/rotatefilehook v0.0.0-20180327172521-2f64f265f58c
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	github.com/subosito/gotenv v1.2.0
	github.com/tidwall/gjson v1.13.0
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/h2non/gock.v1 v1.0.15
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.24.0-alpha.3
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/elastic/go-sysinfo v1.3.0 // indirect
	github.com/elastic/go-ucfg v0.8.3 // indirect
	github.com/elastic/go-windows v1.0.1 // indirect
	github.com/fatih/color v1.7.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/gofrs/uuid v3.2.0+incompatible // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/joeshaw/multierror v0.0.0-20140124173710-69b34d4ec901 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/mailru/easyjson v0.7.1 // indirect
	github.com/mattn/go-colorable v0.0.9 // indirect
	github.com/mattn/go-isatty v0.0.3 // indirect
	github.com/mitchellh/hashstructure v0.0.0-20170116052023-ab25296c0f51 // indirect
	github.com/mitchellh/mapstructure v1.1.2 // indirect
	github.com/pelletier/go-toml v1.2.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/procfs v0.0.9-0.20191208103036-42f6e295b56f // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.0 // indirect
	github.com/spf13/jwalterweatherman v1.0.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/multierr v1.1.1-0.20170829224307-fb7d312c2c04 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20201019141844-1ed22bb0c154 // indirect
	gopkg.in/ini.v1 v1.51.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	howett.net/plist v0.0.0-20181124034731-591f970eefbb // indirect
	sigs.k8s.io/json v0.0.0-20211208200746-9f7c6b3444d2 // indirect
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	github.com/Shopify/sarama => github.com/elastic/sarama v0.0.0-20191122160421-355d120d0970
	github.com/docker/docker => github.com/docker/engine v0.0.0-20191113042239-ea84732a7725
	github.com/docker/go-plugins-helpers => github.com/elastic/go-plugins-helpers v0.0.0-20200207104224-bdf17607b79f
	github.com/dop251/goja => github.com/andrewkroh/goja v0.0.0-20190128172624-dd2ac4456e20
	github.com/elastic/beats/v7 => github.com/elastic/beats/v7 v7.7.1
	github.com/fsnotify/fsevents => github.com/elastic/fsevents v0.0.0-20181029231046-e1d381a4d270
	github.com/fsnotify/fsnotify => github.com/adriansr/fsnotify v0.0.0-20180417234312-c9bbe1f46f1d
	github.com/getkin/kin-openapi => github.com/getkin/kin-openapi v0.67.0
	github.com/google/gopacket => github.com/adriansr/gopacket v1.1.18-0.20200327165309-dd62abfa8a41
	github.com/insomniacslk/dhcp => github.com/elastic/dhcp v0.0.0-20200227161230-57ec251c7eb3 // indirect
	github.com/tonistiigi/fifo => github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c
)

retract [v1.1.4, v1.1.9] // errored versions
