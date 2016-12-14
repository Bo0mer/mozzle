#!/bin/bash -ex

set -ex

if [ ! -f "/etc/init.d/riemann" ]; then
    wget https://aphyr.com/riemann/riemann_0.2.11_all.deb
    sudo dpkg -i riemann_0.2.11_all.deb
fi
if [ -f "/home/vagrant/riemann.config" ]; then
    sudo mv /home/vagrant/riemann.config /etc/riemann/riemann.config
    sudo service riemann restart
fi

