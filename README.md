# Management Network Library

[![GitHub release](https://img.shields.io/github/release/damianoneill/net.svg)](https://github.com/damianoneill/net/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/damianoneill/net)](https://goreportcard.com/report/github.com/damianoneill/net)
[![license](https://img.shields.io/github/license/damianoneill/net.svg)](https://github.com/damianoneill/net/blob/master/LICENSE)
[![Go Docs](http://img.shields.io/badge/docs-GoDoc-blue.svg)](https://pkg.go.dev/github.com/damianoneill/net/v2?tab=overview)

Management Protocol Network implementations in Go

This package includes:

- Client side support of the NETCONF Protocol defined in [(rfc6241)](https://tools.ietf.org/html/rfc6241).
- Client side support for NETCONF Notifications defined in [(rc5277)](https://tools.ietf.org/html/rfc5277).
- GetSchemas and GetSchema from NETCONF Monitoring defined in [(rfc6022)](https://tools.ietf.org/html/rfc6022).
- Client side support of the SNMP Protocol defined in [(rfc3416)](https://tools.ietf.org/html/rfc3416).

The library includes support for the following cross-cutting concerns through dependency injection:

- Logging
- Metrics
- Configuration

The transport layer is externalized from the Library using dependency injection, allowing the user to choose and configure as their specific environment requires. [Go Examples](https://github.com/damianoneill/net/blob/master/v2/netconf/client/example_test.go) are included for demonstration purposes.

The package can be downloaded with the following command, note the v2 in the module path.

```bash
go get -u github.com/damianoneill/net/v2/...
```

## Credits

The implementation of the framing codec in the rfc6242 package has been adapted from an implementation by [Andrew Fort](https://github.com/andaru) - https://github.com/andaru/netconf.

The implementation of the ber encoding has been leveraged from the implementation by [Geoff Garside](http://geoffgarside.co.uk/) - https://github.com/geoffgarside/ber.
