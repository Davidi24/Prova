``
sudo nano /etc/logrotate.d/kupon_server
``

paste the following snippet into the file 
``````
/home/gjramku@nexus.local/kupon_server/log.txt {
        rotate 0
        daily
        missingok
        notifempty
        postrotate
                systemctl restart kupon_tls_server.service
        endscript
}