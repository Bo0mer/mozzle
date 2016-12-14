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

You need is `VirtualBox` and `vagrant` installed and a *proxyless internet
connection*. After that, it's just
a matter of
```
cd mib
vagrant up --provision
```

After successful provisioning, you'll have Grafana available at
http://localhost:3000 at your host machine. From then on, follow `mozzle`'s
guide to show your application metrics in Grafana.
