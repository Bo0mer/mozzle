# mozzle
[![GoDoc](https://godoc.org/github.com/Bo0mer/mozzle?status.svg)](https://godoc.org/github.com/Bo0mer/mozzle)

Pull metrics for Cloud Foundry applications and forward them to Riemann.

## User's guide
```
Usage of mozzle:
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
    	Cloud Foundry password (default "admin")
  -riemann string
    	Address of the Riemann endpoint (default "127.0.0.1:5555")
  -space string
    	Cloud Foundry space (default "rocket")
  -username string
    	Cloud Foundry user (default "admin")
```

Example:
The following command will emit metrics for all applications under the `NASA`
organization, within the `rocket` space.
```
mozzle -api api.bosh-lite.com -org NASA -space rocket
```
