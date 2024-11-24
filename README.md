# autopgo

Autopgo is collection of components that create an automation pipeline
for [profile guided optimization](https://go.dev/doc/pgo),
specifically for [Go](https://go.dev/) applications.

It consists of 3 main components:

* [Scraper](#scraper) - Used to periodically sample Go applications that expose [pprof](https://github.com/google/pprof)
  endpoints via the [net/http/pprof](https://pkg.go.dev/net/http/pprof) package and forward them to the server
  component.
* [Server](#server) - Used to upload and download pprof profiles from the scraper or via the command-line.
* [Worker](#worker) - Used to merge multiple profiles for the same application into a single base profile for use with
  the `go build` command.

See the section below for more information on the individual components

## Components

This section outlines the responsibilities of each component and how their configuration/operation is managed.

### Scraper

The scraper is responsible for performing regular samples of profiles across multiple applications. Typically, it will
run alongside your applications. Depending on the configuration, it will periodically sample your applications for
profiles and forward them to the server component. The scraper can handle sampling many different instances of your
application concurrently.

#### Command

To run the scraper, use the following command:

```shell
autopgo scrape
```

#### Configuration

The `scrape` command accepts a single argument that is contextual depending on the mode specified via the `--mode` flag.
The `mode` flag accepts `file`, `kube` & `nomad` as values, defaulting to `file`.

The `scrape` command also accepts some command-line flags that may also be set via environment variables. They are
described in the table below:

|         Flag          | Environment Variable  |         Default         | Description                                                                              |
|:---------------------:|:---------------------:|:-----------------------:|:-----------------------------------------------------------------------------------------|
|  `--log-level`, `-l`  |  `AUTOPGO_LOG_LEVEL`  |         `info`          | Controls the verbosity of log output, valid values are `debug`, `info`, `warn` & `error` |
|   `--api-url`, `-u`   |   `AUTOPGO_API_URL`   | `http://localhost:8080` | The base URL of the profile server where scraped profiles will be sent                   |
|    `--port`, `-p`     |    `AUTOPGO_PORT`     |         `8080`          | Specifies the port to use for HTTP traffic                                               |
| `--sample-size`, `-s` | `AUTOPGO_SAMPLE_SIZE` |          None           | Specifies the maximum number of targets to profile concurrently                          |
|     `--app`, `-a`     |     `AUTOPGO_APP`     |          None           | Specifies the the application name that profiles will be uploaded for                    |
|  `--frequency`, `-f`  |  `AUTOPGO_FREQUENCY`  |          `60s`          | Specifies the interval between profiling runs                                            |
|  `--duration`, `-d`   |  `AUTOPGO_DURATION`   |          `30s`          | Specifies the amount of time a target will be profiled for                               |
|    `--mode`, `-m`     |    `AUTOPGO_MODE`     |         `file`          | What mode to run the scraper in (file, kube, nomad)                                      |

##### File Mode

When the `mode` flag is set to `file`, the path to a JSON-encoded configuration file is expected as the argument. This
file should be a JSON array of objects describing where the pprof endpoints are exposed.

```json5
[
  {
    // The scheme, host & port combination of the target.
    "address": "http://localhost:5000",
    // The path to the pprof profile endpoint, defaults to /debug/pprof/profile.
    "path": "/debug/pprof/profile"
  }
]
```

When using `file` mode, the file can be updated without restarting the scraper using a `SIGHUP` signal.

##### Kube Mode

When the `mode` flag is set to `kube`, the first argument becomes an optional path to a kubeconfig file. Scraping
targets are then queried directly from the Kubernetes API. To run using an "in-cluster" configuration with the
appropriate RBAC & service account, you can ignore the first argument.

To make your applications discoverable you must set the `autopgo.scrape` and `autopgo.app` labels & the `autopgo.port`
annotation at the pod level. The table below describes each label/annotation supported by the scraper.

|           Key           |    Type    |                Example                 | Required | Description                                                                             |
|:-----------------------:|:----------:|:--------------------------------------:|:--------:|:----------------------------------------------------------------------------------------|
|    `autopgo.scrape`     |   Label    |        `autopgo.scrape: "true"`        |   Yes    | Informs the scraper that this is a scrape target.                                       |
|  `autopgo.scrape.app`   |   Label    |      `autopgo.app: "hello-world"`      |   Yes    | Informs the scraper which application the profile belongs to.                           |
|  `autopgo.scrape.port`  | Annotation |         `autopgo.port: "8080"`         |   Yes    | Allows for specifying the port the application serves pprof endpoints on.               |
|  `autopgo.scrape.path`  | Annotation | `autopgo.path: "/debug/pprof/profile"` |    No    | Allows for specifying the path to the pprof endpoint, defaults to /debug/pprof/profile. |
| `autopgo.scrape.scheme` | Annotation |        `autopgo.scheme: "http"`        |    No    | Informs the scraper whether the endpoint uses HTTP or HTTPS, defaults to HTTP.          |

It's important to note that in `kube` mode an individual scraper per-application is still required. This is done to keep
sampling code simple and maximising uptime for profiling your applications individually.

Below is an example of a Kubernetes deployment that appropriately sets all labels & annotations:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: example-app
  labels:
    app: example-app
spec:
  selector:
    matchLabels:
      app: example-app
  template:
    metadata:
      name: example-app
      labels:
        app: example-app
        autopgo.scrape: "true"
        autopgo.scrape.app: "example-app"
      annotations:
        autopgo.scrape.path: "/debug/pprof/profile"
        autopgo.scrape.port: "8080"
        autopgo.scrape.scheme: "http"
    spec:
      containers:
        - name: example-app
          image: my-image
```

##### Nomad Mode

When running the scraper in `nomad` mode the first argument usually reserved for a configuration file is no longer
required. If running the scraper against Nomad's services API you instead need to set the typical environment variables
for any nomad client, such as `NOMAD_ADDR` etc. One key difference between this mode and [kube mode](#kube-mode) is that
the `autopgo.port` tag is not required as it can be obtained from the service itself.

The scraper will then source targets from Nomad's services API, searching for any services with appropriate tags added
to their specification. The table below describes these tags and provides examples:

|           Key           |                Example                 | Required | Description                                                                             |
|:-----------------------:|:--------------------------------------:|:--------:|:----------------------------------------------------------------------------------------|
|    `autopgo.scrape`     |        `autopgo.scrape: "true"`        |   Yes    | Informs the scraper that this is a scrape target.                                       |
|  `autopgo.scrape.app`   |      `autopgo.app: "hello-world"`      |   Yes    | Informs the scraper which application the profile belongs to.                           |
|  `autopgo.scrape.path`  | `autopgo.path: "/debug/pprof/profile"` |    No    | Allows for specifying the path to the pprof endpoint, defaults to /debug/pprof/profile. |
| `autopgo.scrape.scheme` |        `autopgo.scheme: "http"`        |    No    | Informs the scraper whether the endpoint uses HTTP or HTTPS, defaults to HTTP.          |

As in all other operating modes, a single scraper instance is required per application you wish to scrape. Below is an
example of a Nomad job specification that contains a service with all usable tags:

```hcl
job "example-app   {
  type = "service"
  group "example-app" {
    count = 1

    network {
      port "http" { to = 8080 }
    }

    task "example-app" {
      driver = "docker"

      config {
        image = "my-image:latest"
        ports = ["http"]
      }

      service {
        name = "example-app"
        port = "http"
        tags = [
          "autopgo.scrape=true",
          "autopgo.scrape.app=example-app",
          "autopgo.scrape.path=/debug/pprof/profile",
          "autopgo.scrape.scheme=http"
        ]
      }
    }
  }
}
```

#### Sampling

The sampling behaviour of the scraper is fairly simple. At the interval defined by the `--frequency` flag, a number
of randomly selected targets (up to the maximum defined in the `--sample-size` flag) have their profiling endpoint
called for the duration defined in the `--duration` flag.

These profiles are taken concurrently and streamed to the upstream profile server, whose base URL is defined via the
`--api-url` flag.

### Server

The server component runs as an HTTP server and handles inbound profiles from the [scraper](#scraper). Upon receiving a
profile its validity is checked before being stored within the configured blob storage provider. Once stored, an event
is published onto the configured event bus to notify the [worker](#worker) component that a profile is ready to be
merged.

The server is also where merged profiles can be downloaded for use with the `go build` command.

#### Command

To run the server, use the following command:

```shell
autopgo server
```

#### Configuration

The `server` command accepts a number of command-line flags that may also be set via environment variables. They are
described in the table below:

|         Flag         |    Environment Variable    | Default | Description                                                                                                                                      |
|:--------------------:|:--------------------------:|:-------:|:-------------------------------------------------------------------------------------------------------------------------------------------------|
| `--log-level`, `-l`  |    `AUTOPGO_LOG_LEVEL`     | `info`  | Controls the verbosity of log output, valid values are `debug`, `info`, `warn` & `error`                                                         |
| `--event-writer-url` | `AUTOPGO_EVENT_WRITER_URL` |  None   | Specifies the event bus to use for publishing profile events. See the documentation on [URLs](#url-configuration) for more details               |
|  `--blob-store-url`  |  `AUTOPGO_BLOB_STORE_URL`  |  None   | Specifies the blob storage provider to use for reading & writing profiles.  See the documentation on [URLs](#url-configuration) for more details |
|    `--port`, `-p`    |       `AUTOPGO_PORT`       | `8080`  | Specifies the port to use for HTTP traffic                                                                                                       |

### Worker

The worker is responsible for handling events published by the [server](#server) component that indicate new profiles
have been uploaded. When it is notified of a new profile, it merges it with any existing profile for the application
specified in the event payload. It reads and writes profile data to a configured blob storage provider.

Once merged, the worker publishes its own event to indicate a successful merge. When the worker receives the
notification indicating a successful merge, the uploaded profile is deleted from blob storage.

#### Command

To run the worker, use the following command:

```shell
autopgo worker
```

#### Configuration

The `worker` command accepts a number of command-line flags that may also be set via environment variables. They are
described in the table below:

|         Flag         |    Environment Variable    | Default | Description                                                                                                                                      |
|:--------------------:|:--------------------------:|:-------:|:-------------------------------------------------------------------------------------------------------------------------------------------------|
| `--log-level`, `-l`  |    `AUTOPGO_LOG_LEVEL`     | `info`  | Controls the verbosity of log output, valid values are `debug`, `info`, `warn` & `error`                                                         |
| `--event-writer-url` | `AUTOPGO_EVENT_WRITER_URL` |  None   | Specifies the event bus to use for publishing profile events. See the documentation on [URLs](#url-configuration) for more details               |
| `--event-reader-url` | `AUTOPGO_EVENT_READER_URL` |  None   | Specifies the event bus to use for consuming profile events. See the documentation on [URLs](#url-configuration) for more details                |
|  `--blob-store-url`  |  `AUTOPGO_BLOB_STORE_URL`  |  None   | Specifies the blob storage provider to use for reading & writing profiles.  See the documentation on [URLs](#url-configuration) for more details |
|    `--port`, `-p`    |       `AUTOPGO_PORT`       | `8080`  | Specifies the port to use for HTTP traffic                                                                                                       |
|      `--prune`       |      `AUTOPGO_PRUNE`       |  None   | Specifies the location of the configuration file for [profile pruning](#pruning)                                                                 |

#### Pruning

The `worker` supports pruning pprof profiles via a configuration file allowing multiple regular expressions per
application to determine frames to drop and keep. This may be useful for managing the size of profiles as typically
profiles will increase in size over time and eliminating code paths from profiles you are not interested in optimising
can help keep those profile sizes low.

Larger profiles will typically cause longer build times, so pruning is something to consider as profiles grow.

To utilise pruning, provide the `--prune` flag with a location to a JSON file that describes the regular expressions to
use. An example configuration is below:

```json5
[
  {
    // The name of the application whose profile you want to prune, this should match a value provided to a scraper via
    // the --app flag or AUTOPGO_APP environment variable.
    "app": "example",
    // The pruning rules applied when profiles for the application are merged
    "rules": [
      {
        // Remove all nodes below the node matching the "drop" regular expression.
        "drop": "github.com\/example\/.*",
        // Optionally, keep any nodes that would otherwise be dropped matching the "keep" regular expression.
        "keep": "github.com\/example\/example-dependency"
      }
    ]
  }
]
```

It is recommended to take a backup of your profile using the [download](#download) command prior to applying a new
pruning rule for the first time.

For specific details on how pruning works, see
the [implementation documentation](https://pkg.go.dev/github.com/google/pprof/profile#Profile.Prune).

## Events

The [server](#server) and [worker](#worker) components communicate via events published to and read from an event bus.
This section describes the event structure and possible payloads. All events are JSON-encoded.

### Envelope

Every event is wrapped with an envelope containing metadata about the event and its payload. Below is an example:

```json5
{
  // A UUID unique to the event.
  "id": "e9e23fa8-9d46-4c66-9190-2f64eacd73a2",
  // When the event was published.
  "timestamp": "2024-10-28T12:16:34.964Z",
  // The type of event, denotes the payload structure.
  "type": "profile.merged",
  // The payload contents.
  "payload": {}
}
```

### Payloads

This section describes individual events published by autopgo components.

#### profile.uploaded

Event that indicates a new profile has been uploaded and is staged for merging.

```json5
{
  // The name of the application the profile is for.
  "app": "example-app",
  // The location of the profile in blob storage.
  "profileKey": "example-app/staging/1730075435311"
}
```

#### profile.merged

Event that indicates an uploaded profile has been merged with the base profile and can be deleted.

```json5
{
  // The name of the application the profile is for.
  "app": "example-app",
  // The location of the profile in blob storage.
  "profileKey": "example-app/staging/1730075435311",
  // The location of the base profile.
  "mergedKey": "example-app/default.pgo"
}
```

#### profile.deleted

Event that indicates the merged and any pending profiles have been deleted for an application.

```json5
{
  // The name of the application that has been deleted.
  "app": "example-app"
}
```

### URL Configuration

The [server](#server) and [worker](#worker) components both utilise URLs for configuring access to both blob storage and
event buses. Currently, these URLs support:

* Blob Storage:
    * S3 (or any compatible S3 storage like Minio)
    * Google Cloud Storage
    * Azure Blob Storage
* Event Buses:
    * AWS SNS/SQS
    * Azure Service Bus
    * GCP Pub Sub
    * Apache Kafka
    * NATS

Where supported, messages published to event buses will use ordering/partition keys using the application name for the
profile. Autopgo uses [gocloud.dev](https://gocloud.dev) to provide general access to a variety of cloud providers.
Please see their documentation for [pubsub](https://gocloud.dev/howto/pubsub/) and
[blob](https://gocloud.dev/howto/blob/) to determine the URL string and additional environment variables required to
configure the server & worker components.

## Utilities

This section outlines additional commands used to work with the main autopgo components.

### Upload

The CLI provides an `upload` command that can be used to manually upload a pprof profile to the [server](#server) to be
merged with any existing profiles for that application.

#### Command

To upload a profile, use the following command, specifying the profile location as the only argument:

```shell
autopgo upload profile.pprof
```

#### Configuration

The `upload` command also accepts some command-line flags that may also be set via environment variables. They are
described in the table below:

|        Flag         | Environment Variable |         Default         | Description                                                                              |
|:-------------------:|:--------------------:|:-----------------------:|:-----------------------------------------------------------------------------------------|
| `--log-level`, `-l` | `AUTOPGO_LOG_LEVEL`  |         `info`          | Controls the verbosity of log output, valid values are `debug`, `info`, `warn` & `error` |
|  `--api-url`, `-u`  |  `AUTOPGO_API_URL`   | `http://localhost:8080` | The base URL of the profile server where the specified profile will be sent              |
|    `--app`, `-a`    |    `AUTOPGO_APP`     |          None           | The name of the application the profile belongs to.                                      |

### Download

The CLI provides a `download` command that can be used to download a merged profile from the [server](#server).

#### Command

To download a profile, use the following command, specifying the application name as the only argument:

```shell
autopgo download hello-world
```

#### Configuration

The `upload` command also accepts some command-line flags that may also be set via environment variables. They are
described in the table below:

|        Flag         | Environment Variable |         Default         | Description                                                                              |
|:-------------------:|:--------------------:|:-----------------------:|:-----------------------------------------------------------------------------------------|
| `--log-level`, `-l` | `AUTOPGO_LOG_LEVEL`  |         `info`          | Controls the verbosity of log output, valid values are `debug`, `info`, `warn` & `error` |
|  `--api-url`, `-u`  |  `AUTOPGO_API_URL`   | `http://localhost:8080` | The base URL of the profile server where the specified profile will be sent              |
|  `--output`, `-o`   |   `AUTOPGO_OUTPUT`   |      `default.pgo`      | The location on the local file system to store the downloaded profile.                   |

### List

The CLI provides a `list` command that can be used to list all merged profiles managed by the [server](#server).

#### Command

To list profiles, use the following command:

```shell
autopgo list
```

#### Configuration

The `list` command also accepts some command-line flags that may also be set via environment variables. They are
described in the table below:

|        Flag         | Environment Variable |         Default         | Description                                                                              |
|:-------------------:|:--------------------:|:-----------------------:|:-----------------------------------------------------------------------------------------|
| `--log-level`, `-l` | `AUTOPGO_LOG_LEVEL`  |         `info`          | Controls the verbosity of log output, valid values are `debug`, `info`, `warn` & `error` |
|  `--api-url`, `-u`  |  `AUTOPGO_API_URL`   | `http://localhost:8080` | The base URL of the profile server where the specified profile will be sent              |

### Delete

The CLI provides a `delete` command that can be used to delete a merged profile from the [server](#server).

#### Command

To download a profile, use the following command, specifying the application name as the only argument:

```shell
autopgo delete hello-world
```

#### Configuration

The `delete` command also accepts some command-line flags that may also be set via environment variables. They are
described in the table below:

|        Flag         | Environment Variable |         Default         | Description                                                                              |
|:-------------------:|:--------------------:|:-----------------------:|:-----------------------------------------------------------------------------------------|
| `--log-level`, `-l` | `AUTOPGO_LOG_LEVEL`  |         `info`          | Controls the verbosity of log output, valid values are `debug`, `info`, `warn` & `error` |
|  `--api-url`, `-u`  |  `AUTOPGO_API_URL`   | `http://localhost:8080` | The base URL of the profile server where the specified profile will be sent              |

### Clean

The CLI provides a `clean` command that can be used to remove profiles larger than a specified size or that have not
been updated for a specified duration.

#### Command

To perform a clean, use the following command, specifying either the `--older-than` or `--larger-than` flags (or both).

```shell
autopgo clean --older-than 24h --larger-than 10240
```

#### Configuration

The `clean` command accepts command-line flags that may also be set via environment variables. They are described in the
table below:

|         Flag          | Environment Variable  |         Default         | Description                                                                              |
|:---------------------:|:---------------------:|:-----------------------:|:-----------------------------------------------------------------------------------------|
|  `--log-level`, `-l`  |  `AUTOPGO_LOG_LEVEL`  |         `info`          | Controls the verbosity of log output, valid values are `debug`, `info`, `warn` & `error` |
|   `--api-url`, `-u`   |   `AUTOPGO_API_URL`   | `http://localhost:8080` | The base URL of the profile server where the specified profile will be sent              |
| `--older-than`, `-d`  | `AUTOPGO_OLDER_THAN`  |          None           | How long a profile must not have been updated for to be eligible for cleaning            |
| `--larger-than`, `-s` | `AUTOPGO_LARGER_THAN` |          None           | The minimum size (in bytes) a profile must be to be eligible for cleaning                |
