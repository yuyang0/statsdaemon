statsdaemon
==========

Port of Etsy's statsd (https://github.com/etsy/statsd), written in Go (originally based
on [amir/gographite](https://github.com/amir/gographite)).

Supports

* Timing (with optional percentiles)
* Counters (positive and negative with optional sampling)
* Gauges (including relative operations)
* Sets

Note: Only integers are supported for metric values.

[![Build Status](https://secure.travis-ci.org/bitly/statsdaemon.png)](http://travis-ci.org/bitly/statsdaemon)

Installing
==========

### Binary Releases
Pre-built binaries for darwin and linux.

### Current Stable Release: `v0.7.1`
* [statsdaemon-0.7.1.darwin-amd64.go1.4.2.tar.gz](https://github.com/bitly/statsdaemon/releases/download/v0.7.1/statsdaemon-0.7.1.darwin-amd64.go1.4.2.tar.gz)
* [statsdaemon-0.7.1.linux-amd64.go1.4.2.tar.gz](https://github.com/bitly/statsdaemon/releases/download/v0.7.1/statsdaemon-0.7.1.linux-amd64.go1.4.2.tar.gz)

### Older Releases
* [statsdaemon-0.6-alpha.darwin-amd64.go1.3.tar.gz](https://github.com/bitly/statsdaemon/releases/download/v0.6-alpha/statsdaemon-0.6-alpha.darwin-amd64.go1.3.tar.gz)
* [statsdaemon-0.6-alpha.linux-amd64.go1.3.tar.gz](https://github.com/bitly/statsdaemon/releases/download/v0.6-alpha/statsdaemon-0.6-alpha.linux-amd64.go1.3.tar.gz)
* [statsdaemon-0.5.2-alpha.linux-amd64.go1.1.1.tar.gz](https://github.com/bitly/statsdaemon/releases/download/v0.5.2-alpha/statsdaemon-0.5.2-alpha.linux-amd64.go1.1.1.tar.gz)

### Building from Source
```
git clone https://github.com/bitly/statsdaemon
cd statsdaemon
go get github.com/bmizerany/assert #for tests
go build
```
## Opren-falcon
this fork supports open-falcon as a backend, the metric name has the following format

    part1._e_endpoint.part2/tag1=val1,tag2=val2

this format mainly used to support endpoint and tags in open-falcon, the first
part starts with `_e_` is the endpoint name, you can use the command line
argument to change the default endpoint prefix to other value you like, the last
part seprate by '/' is the tags.

Command Line Options
====================

```
Usage of ./statsdaemon:
  -address=":8125": UDP service address
  -debug=false: print statistics sent to graphite
  -delete-gauges=true: don't send values to graphite for inactive gauges, as opposed to sending the previous value
  -endpoint-prefix="_e_": prefix of endpoint name
  -flush-interval=10: Flush interval (seconds)
  -backend="open-falcon": which backend(graphite or open-falcon) you want to use
  -graphite="127.0.0.1:2003": Graphite service address (or - to disable)
  -open-falcon="127.0.0.1:1988": the listen address of open-falcon's agent
  -max-udp-packet-size=1472: Maximum UDP packet size
  -percent-threshold=[]: percentile calculation for timers (0-100, may be given multiple times)
  -persist-count-keys=60: number of flush-intervals to persist count keys
  -postfix="": Postfix for all stats
  -prefix="": Prefix for all stats
  -receive-counter="": Metric name for total metrics received per interval
  -tcpaddr="": TCP service address, if set
  -version=false: print version string
```
