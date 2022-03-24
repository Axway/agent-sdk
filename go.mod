module github.com/Axway/agent-sdk

go 1.16

require (
	github.com/elastic/beats/v7 v7.7.1
	github.com/emersion/go-sasl v0.0.0-20211008083017-0b9dcfb154ac
	github.com/emersion/go-smtp v0.15.0
	github.com/emicklei/proto v1.9.2
	github.com/fsnotify/fsnotify v1.5.1
	github.com/gabriel-vasile/mimetype v1.4.0
	github.com/getkin/kin-openapi v0.76.0
	github.com/golang-jwt/jwt v3.2.2+incompatible
	github.com/google/uuid v1.3.0
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/sirupsen/logrus v1.8.1
	github.com/snowzach/rotatefilehook v0.0.0-20220211133110-53752135082d
	github.com/spf13/cobra v1.4.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.1
	github.com/subosito/gotenv v1.2.0
	github.com/tidwall/gjson v1.14.0
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f
	google.golang.org/grpc v1.45.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/h2non/gock.v1 v1.1.2
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.22.7
)

replace (
	github.com/Shopify/sarama => github.com/elastic/sarama v1.19.0
	github.com/dop251/goja => github.com/andrewkroh/goja v0.0.0-20190128172624-dd2ac4456e20
	github.com/elastic/beats/v7 => github.com/elastic/beats/v7 v7.7.1
	github.com/fsnotify/fsevents => github.com/elastic/fsevents v0.0.0-20181029231046-e1d381a4d270
	github.com/getkin/kin-openapi v0.76.0 => github.com/getkin/kin-openapi v0.67.0
)

retract ( // errored versions
	v1.1.16
	[v1.1.4, v1.1.9]
)
