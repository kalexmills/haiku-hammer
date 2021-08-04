cp init-script.sh /etc/init.d/haiku-hammer
cp config.yaml /etc/haikuhammer/config.yaml
systemctl daemon-reload
service haiku-hammer restart