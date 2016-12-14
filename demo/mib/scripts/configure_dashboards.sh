#!/bin/bash -ex

set -ex

for dashboard in /home/vagrant/*.json; do
    dashboard_json=$(cat $dashboard)
    curl -X POST \
        -H "Content-Type: application/json" \
        -d '{"dashboard":'"${dashboard_json}"',"overwrite": true}' \
        -v \
        http://admin:admin@127.0.0.1:3000/api/dashboards/db
done
