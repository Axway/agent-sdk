module github.com/Axway/agent-sdk

go 1.16

require (
	github.com/elastic/beats/v7 v7.17.5
	github.com/elastic/elastic-agent-client/v7 v7.0.0-20220607160924-1a71765a8bbe // indirect
	github.com/elastic/go-licenser v0.4.1 // indirect
	github.com/elastic/go-sysinfo v1.8.1 // indirect
	github.com/elastic/go-ucfg v0.8.6 // indirect
	github.com/emersion/go-sasl v0.0.0-20211008083017-0b9dcfb154ac
	github.com/emersion/go-smtp v0.15.0
	github.com/emicklei/proto v1.9.2
	github.com/fsnotify/fsnotify v1.5.4
	github.com/gabriel-vasile/mimetype v1.4.0
	github.com/getkin/kin-openapi v0.76.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.3.0
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/jcchavezs/porto v0.4.0 // indirect
	github.com/lestrrat-go/jwx v1.2.25
	github.com/mitchellh/hashstructure v1.1.0 // indirect
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pelletier/go-toml/v2 v2.0.2 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/sirupsen/logrus v1.8.1
	github.com/snowzach/rotatefilehook v0.0.0-20220211133110-53752135082d
	github.com/spf13/cobra v1.5.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.8.0
	github.com/subosito/gotenv v1.4.0
	github.com/tidwall/gjson v1.14.0
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80
	go.elastic.co/apm v1.15.0 // indirect
	go.elastic.co/ecszap v1.0.1 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/crypto v0.0.0-20220622213112-05595931fe9d // indirect
	golang.org/x/net v0.0.0-20220708220712-1185a9018129
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
	golang.org/x/tools v0.1.11 // indirect
	google.golang.org/genproto v0.0.0-20220714211235-042d03aeabc9 // indirect
	google.golang.org/grpc v1.48.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/h2non/gock.v1 v1.1.2
	gopkg.in/ini.v1 v1.66.6 // indirect
	gopkg.in/yaml.v3 v3.0.1
	howett.net/plist v1.0.0 // indirect
	k8s.io/apimachinery v0.22.7
)

replace (
	github.com/Shopify/sarama => github.com/elastic/sarama v1.19.1-0.20210823122811-11c3ef800752
	github.com/getkin/kin-openapi => github.com/getkin/kin-openapi v0.67.0
)

retract ( // errored versions
	[v1.1.21, v1.1.23]
	v1.1.16
	[v1.1.4, v1.1.9]
)
