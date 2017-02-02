# mozzle
[![GoDoc](https://godoc.org/github.com/Bo0mer/mozzle?status.svg)](https://godoc.org/github.com/Bo0mer/mozzle)

Pull metrics for Cloud Foundry applications and forward them to Riemann.

This repo provides two functionalities - a `mozzle` Go package and a mozzle
command-line tool.

Package mozzle provides an API for monitoring infrastructure metrics of
Cloud Foundry applications and emitting them to a 3rd party monitoring system.

The `mozzle` command-line tool emits metrics for Cloud Foundry applications
to a specified Riemann instance. The rest of this document describes its usage.

Before reading futher, make sure you have a running Riemann instance. If you want
just to try out `mozzle`, refer to [this guide](https://github.com/Bo0mer/mozzle/tree/master/demo/mib/) how to setup one in a minute.

## User's guide
If you want to monitor all applications under your current Cloud Foundry target,
as set with the CF CLI, you can do the following.
```
mozzle -use-cf-cli-target
```

If you want to explicitly specify the monitored target, you can do that too.
```
# This example assumes that you have exported the necessary env variables
mozzle -api https://api.bosh-lite.com -access-token $CF_ACCESS_TOKEN -refresh-token $CF_REFRESH_TOKEN -org NASA -space rocket
```

If you do not want to deal with access and refresh tokens, you can provide plain
username and password.
```
mozzle -api https://api.bosh-lite.com -username admin -password admin -org NASA -space rocket
```

And if your Cloud Foundry has invalid TLS certificate for some reason, you can skip its verification.
```
mozzle -insecure -api https://api.bosh-lite.com -username admin -password admin -org NASA -space rocket
```

Following is a full list of supported command-line flag arguments.
```
Usage of mozzle:
  -access-token string
    	Cloud Foundry OAuth2 token; either token or username and password must be provided
  -api string
    	Address of the Cloud Foundry API (default "https://api.bosh-lite.com")
  -events-queue-size int
    	Queue size for outgoing events (default 256)
  -events-ttl float
    	TTL for emitted events (in seconds) (default 30)
  -insecure
    	Please, please, don't!
  -org string
    	Cloud Foundry organization (default "NASA")
  -password string
    	Cloud Foundry password; usage is discouraged - see token option instead
  -refresh-token string
    	Cloud Foundry OAuth2 refresh token; to be used with the token flag
  -riemann string
    	Address of the Riemann endpoint (default "127.0.0.1:5555")
  -space string
    	Cloud Foundry space (default "rocket")
  -use-cf-cli-target
    	Use CF CLI's current configured target
  -username string
    	Cloud Foundry user; usage is discouraged - see token option instead
  -v	Report mozzle version
  -version
    	Report mozzle version
```

### Demo usage
This repo brings a [vagrant](https://www.vagrantup.com/) automation that will setup a VM ready for
showing your application metrics. For more info on settin it up, refer to its
[README](https://github.com/Bo0mer/mozzle/tree/master/demo/mib/) file.
