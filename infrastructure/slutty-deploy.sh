#!/bin/bash -e

go build ../
ssh -p2222 root@scp.click systemctl stop schttp
scp -P2222 schttp root@scp.click:
ssh -p2222 root@scp.click systemctl start schttp
sleep 1
ssh -p2222 root@scp.click systemctl status schttp