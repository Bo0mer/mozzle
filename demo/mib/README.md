mib
===
## mozzle in a box 

![mozzle metrics in a Grafana dashboard](https://raw.githubusercontent.com/Bo0mer/mozzle/master/demo/mib/misc/mozzle_demo_dashboard.png)

This vagrant setup provides infrastructure for monitoring Cloud Foundry applications
via [mozzle](https://github.com/Bo0mer/mozzle).

### What's inside the box
After provisioning the VM, you'll have the following components installed 
and configured - thus ready for use.
* Grafana - used for visualizing collected data
* InfluxDB - used for storing collected data
* Riemann - used for routing data to different sources, incl InfluxDB.

### Running it

You need is `VirtualBox` and `vagrant` installed and a **proxyless internet
connection**. After that, it's just
a matter of
```
cd mib
vagrant up --provision
```

After successful provisioning, you'll have Grafana available at
[http://localhost:3000](http://localhost:3000/) at your host machine.
The default credentials are `admin:admin`.

After login, you see a list of predefined dashboards — namely Overview, Events
and HTTP Statistics. Your application metrics will be visible there.

The last step is to instruct mozzle to start pulling metrics. It operates on CF
 space level and has option to derive the monitored target from the CF CLI.

```
$ cf target
API endpoint:   https://api.run.pivotal.io (API version: 2.69.0)
User:           [my-email]@gmail.com
Org:            NASA
Space:          rocket
$ mozzle -use-cf-cli-target
```

The execution should block and metrics should start to appear in the Grafana dashboards.
You should see something like the picture above.

