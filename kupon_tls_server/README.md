# Kupon TLS Server

TLS server certificates:
```
openssl req -new -nodes -x509 -out certs/server.pem -keyout certs/server.key -days 3650 -subj "/C=DE/ST=NRW/L=Earth/O=Random Company/OU=IT/CN=www.random.com/emailAddress=$1"
```
Shembull konkret
``
openssl req -new -nodes -x509 -out server.pem -keyout server.key -days 3650 -subj "/C=AL/ST=TR/L=Tirane/O=AED/OU=AED/CN=aed.al/emailAddress=info@aed.al"
``


If there are changes in `systemd` script run:
```bash
systemctl daemon-reload
```

To enable the service after installation:
```bash
systemctl enable kupon_tls_server.service
```

To stop the service:
```bash
systemctl stop kupon_tls_server.service
```

To start the service:
```bash
systemctl start kupon_tls_server.service
```

To restart the service:
```bash
systemctl restart kupon_tls_server.service
```

To restart the service:
```bash
systemctl restart kupon_tls_server.service
```
To check status of the service:
```bash
systemctl status kupon_tls_server.service
```

To see the STDOUT of the process, run:
```bash
journalctl -f -u kupon_tls_server.service
```