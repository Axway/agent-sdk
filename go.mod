module github.com/Axway/agent-sdk

go 1.26.0

require (
	github.com/emicklei/proto v1.14.3
	github.com/fsnotify/fsnotify v1.10.1
	github.com/getkin/kin-openapi v0.149.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0
	github.com/invopop/yaml v0.3.1
	github.com/lestrrat-go/jwx/v2 v2.1.7
	github.com/opentracing/opentracing-go v1.2.0
	github.com/shopspring/decimal v1.4.0
	github.com/sirupsen/logrus v1.10.2
	github.com/snowzach/rotatefilehook v0.0.0-20220211133110-53752135082d
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.12.1
	github.com/subosito/gotenv v1.6.0
	github.com/swaggest/go-asyncapi v0.8.1
	github.com/tomnomnom/linkheader v0.0.0-20250811210735-e5fe3b51442e
	golang.org/x/net v0.59.0
	golang.org/x/text v0.42.0
	google.golang.org/grpc v1.83.2
	google.golang.org/protobuf v1.36.12
	gopkg.in/h2non/gock.v1 v1.1.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.1 // indirect
	github.com/go-openapi/jsonpointer v1.0.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.4 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/oasdiff/yaml v0.1.1 // indirect
	github.com/oasdiff/yaml3 v0.0.14 // indirect
	github.com/pelletier/go-toml/v2 v2.4.3 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260908043556-f8649ddbbfe6 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)

retract ( // errored versions
	[v1.1.35, v1.1.36]
	[v1.1.21, v1.1.23]
	v1.1.16
	[v1.1.4, v1.1.9]
)
