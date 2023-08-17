#!/bin/bash

chmod a+x kupon_tls_server

cp kupon_tls_server.service /etc/systemd/system/kupon_tls_server.service
systemctl daemon-reload
systemctl enable kupon_tls_server.service
systemctl start kupon_tls_server.service
