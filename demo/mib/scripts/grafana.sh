#!/bin/bash -ex

set -ex

if [ ! -f "/etc/init.d/grafana-server" ]; then
	wget https://grafanarel.s3.amazonaws.com/builds/grafana_4.0.2-1481203731_amd64.deb
	sudo dpkg -i grafana_4.0.2-1481203731_amd64.deb
	rm grafana_4.0.2-1481203731_amd64.deb

	sudo service grafana-server restart

    curl -X POST \
        -H "Content-Type: application/json" \
        -d '{"name":"InfluxDB","type":"influxdb","url":"http://localhost:8086","access":"proxy","jsonData":{},"isDefault":true,"database":"metrics"}' \
        -v \
        http://admin:admin@127.0.0.1:3000/api/datasources
fi

