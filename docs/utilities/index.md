# Amplify Agents SDK Utilities

In addition to the core features provided by Amplify Agents SDK around API discovery and traceability, it also provide some of the helpful utilities that developers can use while building agent as and where needed.

# REST API HTTP Client

The agent provides support for creating HTTP clients with *api* package that can be used for making HTTP request and processing the response. The HTTP Client can be initialized with TLS security (*config.TLSConfig*) and can use proxy.

The package provides *NewClient()* method to create a new HTTP client. This method take an interface of *config.TLSConfig* and proxy URL as argument. Below is a sample of creating HTTP Client with default TLS config and no proxy configured

```
 tlsConfig := config.NewTLSConfig()
 apiClient := api.NewClient(tlsConfig, "")
```

The HTTP client interface provides *Send()* method which takes an object of *api.Request* struct. The *api.Request* identifies the HTTP request to be sent and holds following properties

- Method : Identifies the HTTP method to be used for the request. Supported values are "GET", "PUT", "POST" and "DELETE"
- URL : Specifies the target HTTP endpoint where the request will be sent
- QueryParams: : Map of key-value pairs that will be added as query parameter to HTTP request
- Headers : Map of key-value pairs that will be added as request headers
- Body: Represents the body that is used for PUT and POST requests.

The *Send()* method of the HTTP client returns an object of type *api.Response* which holds following properties

- Code : Represents the HTTP response code returned for HTTP request
- Header : Map of key-value pairs that represents HTTP response headers
- Body : Represents the body returned as HTTP response.

Below is a sample of constructing the request, use the HTTP client to send the request and receive response

```
    query := map[string]string{
        "param1": "value1",
        "param2": "value2",
    }

    header := map[string]string{
        "Authorization": "Bearer " + token,
        "x-some-other-header": "header-value",
    }

    request := api.Request{
        Method: api.GET,
        URL:    "http://someURL",
        QueryParams: queryParams,
        Headers: header
    }

    response, err := apiClient.Send(request)

    log.Debug("Status : " + strconv.Itoa(response.Code))
    log.Debug("Body : " + string(response.Body))
```

# Cache

The Amplify Agents SDK provides an in-memory cache using *cache* package that developers can use to store items that are frequently used for faster access. The cache stores items based on key and optionally secondary key if needed by the implementation. The items can be queried using either key or secondary key assigned to the item. The Amplify Agents SDK exposes the following interface which that describes the methods provided by *cache*

```
type Cache interface {
 Get(key string) (interface{}, error)
 GetBySecondaryKey(secondaryKey string) (interface{}, error)
 GetKeys() []string
 HasItemChanged(key string, data interface{}) (bool, error)
 HasItemBySecondaryKeyChanged(secondaryKey string, data interface{}) (bool, error)
 Set(key string, data interface{}) error
 SetWithSecondaryKey(key string, secondaryKey string, data interface{}) error
 SetSecondaryKey(key string, secondaryKey string) error
 Delete(key string) error
 DeleteBySecondaryKey(secondaryKey string) error
 DeleteSecondaryKey(secondaryKey string) error
 Flush()
 Save(path string) error
 Load(path string) error
}
```

Below is an sample demonstrating the creation of cache, storing items and then querying them

```
objCache = cache.New()

objCache.Set("key", object)

objCache.SetWithSecondaryKey("key-1", "sub-key-1", object1)

objCache.Set("key-2", object2)

objCache.SetSecondaryKey("key-2", "sub-key-2")

....

obj, err := objCache.Get("key"

obj, err := objCache.Get("key-1")
obj, err := objCache.GetBySecondaryKey("sub-key-1")


obj, err := objCache.Get("key-2")
obj, err := objCache.GetBySecondaryKey("sub-key-2")

err := objCache.Delete("key)

err := objCache.DeleteBySecondaryKey("sub-key-1)
```

The cache store the has of the item, so implementation can validate if the object has changed. For example

```
obj.prop = 111
objCache.Set("key", obj)

obj.prop = 222
isChanged, err := objCache.HasItemChanged("key", obj)
```

# Health checker

The Amplify Agents SDK implements a health check service that gets initialized during agent initialization. The service calls the list of registered callbacks to perform the check on the corresponding service. The service also exposed an endpoint over port 8080, that users can use to make HTTP based call to verify health check of the agent overall and of individual components (registered health check callbacks). The health check endpoint port is configurable using *status.port* config.

The health check callback function should be of CheckStatus type (described below). The callback implementation can set the status of the component/service by creating an object of type *healthcheck.Status* and setting *Result* field to represent the current status of the component. The status level can be "OK" or "FAIL"

```
type CheckStatus func(name string) *Status
```

The Amplify Agents SDK provides *RegisterHealthCheck()* method in *healthcheck* package to register the callback function which will be invoked at an interval. The interval is configurable using config *status.healthCheckInterval* which can be set in yaml or overridden using environment variable.

Following is a sample of callback registration and callback implementation

```
func run() error {
    healthcheck.RegisterHealthcheck("API Manager", "apimanager", healthcheck)
}


func (c *v7Client) healthcheck(name string) (status *hc.Status) {
    ... 
    // perform the check on component 
    ...
    return &Status {
        Result: healthcheck.FAIL
        Details "Error description"
    }
}
```

# Logging

The Amplify Agents SDK utilizes [logrus](https://github.com/sirupsen/logrus/blob/master/README.md) and provides a structured logger that can be used by agent implementation to have unified logging. The Amplify Agents SDK setup the logger during the initialization. Below are the list of configuration properties that Amplify Agents SDK provides to configure the logger. The logger supports both stdout and file outputs and can log in line or JSON format. The logger provided by Amplify Agents SDK supports log rotation based on size and can keep the configured number of backups of old log files.

| Environment variable      | YAML                      | Description                                                                                                                                                                          |
| ------------------------- | ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| LOG_LEVEL                 | log.level                 | The log level for output messages (debug, info, warn, error)                                                                                                                         |
| LOG_FORMAT                | log.format                | The format to print log messages (json, line, package)                                                                                                                               |
| LOG_OUTPUT                | log.output                | The output for the log lines (stdout, file, both)                                                                                                                                    |
| LOG_MASKEDVALUES          | log.maskedValues          | Comma-separated list of key words to identify within the agent config and used to mask its corresponding sensitive data. Key words are matched by whole words and are case sensitive |
| LOG_FILE_NAME             | log.file.name             | The name of the log files                                                                                                                                                            |
| LOG_FILE_PATH             | log.file.path             | The path (relative or absolute) to save logs files, if output type file or both                                                                                                      |
| LOG_FILE_ROTATEEVERYBYTES | log.file.rotateeverybytes | The max size, in bytes that a log file can grow to                                                                                                                                   |
| LOG_FILE_KEEPFILES        | log.file.keepfiles        | The max number of log file backups to keep                                                                                                                                           |
| LOG_FILE_CLEANBACKUPS     | log.file.cleanbackups     | The max age of a backup file, in days                                                                                                                                                |

The *log* package provides following methods that agents can call to log

```
func Error(args ...interface{})
func Errorf(format string, args ...interface{})

func Debug(args ...interface{})
func Debugf(format string, args ...interface{})

func Info(args ...interface{})
func Infof(format string, args ...interface{})

func Warn(args ...interface{})
func Warnf(format string, args ...interface{})
```

Some of the samples of logging using Amplify Agents SDK logger

```
log.Info("Some thing to log")
log.Infof("Log entry with format, %s", "additional log ~~message")

log.Debugf("No changes detected in the API %s", *azAPI.Name)

log.Trace("I got here in the code")

log.Errorf("Error in processing : %s", err.Error())

log.Warnf("Config not found: %s", missingItem)
```
