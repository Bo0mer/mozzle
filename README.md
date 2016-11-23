# mozzle
Pull metrics for Cloud Foundry applications and forward them to statsd

## User's guide
```
Usage of mozzle:
  -api string
    	Address of the Cloud Foundry API
  -app-guid string
    	Cloud Foundry application GUID
  -interval int
    	Interval (in seconds) between reports (default 5)
  -password string
    	Cloud Foundry password (default "admin")
  -statsd string
    	Address of the statsd endpoint (default "127.0.0.1:8125")
  -username string
    	Cloud Foundry user (default "admin")
```

Example:
```
mozzle -api api.bosh-lite.com -app-guid $(cf app my-fancy-app --guid)
```
