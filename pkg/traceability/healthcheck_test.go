package traceability

// func TestExecute(t *testing.T) {
// 	job := condorHealthCheckJob{
// 		protocol: "tcp",
// 		host:     "somehost.com:543",
// 		proxyURL: "",
// 		timeout:  time.Second * 1,
// 	}
// 	// hc not okay
// 	err := job.Execute()
// 	assert.Nil(t, err)
// }

// func TestReady(t *testing.T) {
// 	job := condorHealthCheckJob{
// 		protocol: "tcp",
// 		host:     "somehost.com:543",
// 		proxyURL: "",
// 		timeout:  time.Second * 1,
// 	}
// 	// hc not okay
// 	ready := job.Ready()
// 	assert.False(t, ready)
// }

// func TestJobStatus(t *testing.T) {
// 	os.Setenv("HTTP_CLIENT_TIMEOUT", "1s")

// 	testCases := []struct {
// 		name     string
// 		protocol string
// 		proxy    string
// 		errStr   string
// 	}{
// 		{
// 			name:     "TCP no Proxy",
// 			protocol: "tcp",
// 			proxy:    "",
// 			errStr:   "connection failed",
// 		},
// 		{
// 			name:     "TCP bad Proxy URL",
// 			protocol: "tcp",
// 			proxy:    "socks5://host:\\//localhost:1080",
// 			errStr:   "proxy could not be parsed",
// 		},
// 		{
// 			name:     "TCP bad Proxy Protocol",
// 			protocol: "tcp",
// 			proxy:    "sock://localhost:1080",
// 			errStr:   "could not setup proxy",
// 		},
// 		{
// 			name:     "TCP good Proxy",
// 			protocol: "tcp",
// 			proxy:    "socks5://localhost:1080",
// 			errStr:   "connection failed",
// 		},
// 		{
// 			name:     "HTTPS no Proxy",
// 			protocol: "https",
// 			proxy:    "",
// 			errStr:   "connection failed",
// 		},
// 	}

// 	for _, test := range testCases {
// 		t.Run(test.name, func(t *testing.T) {
// 			traceCfg = &Config{
// 				Proxy: ProxyConfig{
// 					URL: test.proxy,
// 				},
// 			}
// 			job := condorHealthCheckJob{
// 				protocol: test.protocol,
// 				host:     "somehost.com:543",
// 				proxyURL: test.proxy,
// 				timeout:  time.Second * 1,
// 			}

// 			err := job.Status()
// 			assert.NotNil(t, err)
// 			assert.Contains(t, err.Error(), test.errStr)
// 		})
// 	}
// }
