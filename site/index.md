# Simple Proxy

Simple Proxy is exactly what it says in title - Simple Proxy.

Simple Proxy allows you to run TCP proxy capable of routing the connection based on HTTP Host header or SNI hostname in TLS connection.

## Install

Pre-built binaries are available from [releases][] page. You can also install simple-proxy using `go get`.

Using `go get`:

```sh
go get -u github.com/Gufran/simple-proxy
```

[releases]: https://github.com/Gufran/simple-proxy/releases


## Configuration

Proxy configuration is written in HCL. Following parameters are supported in the configuration file:

 - `listen` (`block`): Configures a local listener. `listen` takes one label which must be a properly formatted listener address in `host:port` or `:port` format.
   Multiple `listen` blocks can be used to configure multiple listeners. Following attributes are supported inside `listen` block.
    - `proxy_to`: Destination address for proxy
    - `route`: Routing configuration for the proxy.

??? note "Example"
    ```hcl
    # Listen on port 8765 on loopback interface
    listen "127.0.0.1:8765" {
      // ...
    }

    # Listen on port 8765 on all interfaces
    listen ":8765" {
      // ...
    }

    # Same port can be used multiple times
    listen ":8765" {
      // ...
    }
    ```

 - `proxy_to` (`string`, `optional`): `proxy_to` is supported at root level of `listener` block and configures a direct proxy to destination.
   Value of `proxy_to` must adhere to `host:port` format.

??? note "Example"
    ```hcl
    listen ":8765" {
      proxy_to = "my-domain.com:7654"
    }
    ```

 - `route` (`block`, `optional`): `route` block configures a routing decision. Multiple `route` blocks can be specified on a listener to configure multiple routes.
   Following attributes are supported inside `route` block:
    - `host`: Host header to match for HTTP/1 request routing
    - `sni`: SNI header to match for TLS connection routing
    - `to`: Proxy destination address. Must be in `host:port` format


??? note "Example"
    ```hcl
    listen ":8765" {
      route {
        host = "my-domain.com"
        to = "other-domain.com"
      }
      
      route {
        sni = "secure-domain.com"
        to = "other-secure-domain.com"
      }
      
      route {
        host = "my-domain.com"
        sni = "other-domain.com"
        to = "same-domain.com"
      }
    }
    ```

!!! note
    Only one of `proxy_to` or `route` can be specified, not both. At least one of the attributes must be configured on each listener.

