module git.ecd.axway.int/apigov/apic_agents_sdk

go 1.13

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	github.com/Shopify/sarama => github.com/elastic/sarama v0.0.0-20191122160421-355d120d0970
	github.com/docker/docker => github.com/docker/engine v0.0.0-20191113042239-ea84732a7725
	github.com/docker/go-plugins-helpers => github.com/elastic/go-plugins-helpers v0.0.0-20200207104224-bdf17607b79f
	github.com/dop251/goja => github.com/andrewkroh/goja v0.0.0-20190128172624-dd2ac4456e20
	github.com/fsnotify/fsevents => github.com/elastic/fsevents v0.0.0-20181029231046-e1d381a4d270
	github.com/fsnotify/fsnotify => github.com/adriansr/fsnotify v0.0.0-20180417234312-c9bbe1f46f1d
	github.com/google/gopacket => github.com/adriansr/gopacket v1.1.18-0.20200327165309-dd62abfa8a41
	github.com/insomniacslk/dhcp => github.com/elastic/dhcp v0.0.0-20200227161230-57ec251c7eb3 // indirect
	github.com/tonistiigi/fifo => github.com/containerd/fifo v0.0.0-20190816180239-bda0ff6ed73c
)

require (
	git.ecd.axway.int/apigov/service-mesh-agent v0.0.0-20200421210019-84ac6b925933
	github.com/elastic/beats v7.6.2+incompatible
	github.com/elastic/go-sysinfo v1.3.0 // indirect
	github.com/elastic/go-ucfg v0.7.0 // indirect
	github.com/getkin/kin-openapi v0.9.0
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/google/uuid v1.1.1
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1
	github.com/tidwall/gjson v1.6.0
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/sys v0.0.0-20200615200032-f1bc736245b1 // indirect
	golang.org/x/tools v0.0.0-20200619023621-037be6a06566 // indirect
	gopkg.in/h2non/gock.v1 v1.0.15
	gopkg.in/yaml.v2 v2.3.0 // indirect
)
