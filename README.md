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
profiles and forward them to the server component. The scraper can handle sampling many different applications
concurrently, keeping your profiles separated.

#### Command

To run the scraper, use the following command:

```shell
autopgo scrape config.json
```

#### Configuration

The `scrape` command accepts a single argument that is a path to a JSON-encoded configuration file that describes the
sampling behaviour and the targets available to be scraped. Below is an example configuration:

```json5
{
  // The maximum number of targets to sample at once.
  "sampleSize": 3,
  // How long targets should be profiled for, in seconds.
  "profileDuration": 30,
  // How frequently targets should be profiled, in seconds.
  "scrapeFrequency": 300,
  // Endpoints that will profile applications.
  "targets": [
    {
      // The application the profile will belong to.
      "app": "example-app",
      // The full address pointing to the target's profiling endpoint.
      "address": "http://localhost:5000/debug/pprof/profile"
    }
  ]
}
```

The `scrape` command also accepts some command-line flags that may also be set via environment variables. They are
described in the table below:

|        Flag         | Environment Variable |         Default         | Description                                                                              |
|:-------------------:|:--------------------:|:-----------------------:|:-----------------------------------------------------------------------------------------|
| `--log-level`, `-l` | `AUTOPGO_LOG_LEVEL`  |         `info`          | Controls the verbosity of log output, valid values are `debug`, `info`, `warn` & `error` |
|  `--api-url`, `-u`  |  `AUTOPGO_API_URL`   | `http://localhost:8080` | The base URL of the profile server where scraped profiles will be sent                   |
|   `--port`, `-p`    |    `AUTOPGO_PORT`    |         `8080`          | Specifies the port to use for HTTP traffic                                               |

#### Sampling

The sampling behaviour of the scraper is fairly simple. At the interval defined in the `scrapeFrequency` field, a number
of randomly selected targets (up to the maximum defined in the `sampleSize` field) have their profiling endpoint called
for the duration defined in the `profileDuration` field.

These profiles are taken concurrently and streamed to the upstream profile server, whose base URL is defined via the
`apiUrl` field.

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
