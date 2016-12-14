#!/bin/bash -ex

set -ex

if [ ! -f "/etc/init.d/influxdb" ]; then
    wget https://dl.influxdata.com/influxdb/releases/influxdb_1.1.1_amd64.deb
    sudo dpkg -i influxdb_1.1.1_amd64.deb
    rm influxdb_1.1.1_amd64.deb
fi

if [ -f "/home/vagrant/influxdb.conf" ]; then
    sudo mv /home/vagrant/influxdb.conf /etc/influxdb/influxdb.conf
    sudo service influxdb restart

    echo "create database metrics;" | influx
fi


