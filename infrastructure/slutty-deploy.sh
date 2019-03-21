#!/bin/bash -e

export GO111MODULE=on
go build ../

# as go does magic things to executables - we cannot overwrite the file while
# it is running - but we can move it out of the way (which will properly save us one day anyway)
ssh -p2222 root@scp.click 'mv /root/schttp $(mktemp schttp-update-XXXXX)'

# Move the new executable
scp -P2222 schttp root@scp.click:schttp

# reload makes systemd HUP the old process - forwarding listeners to the new 
# letting existing tranfers complete in the old process
ssh -p2222 root@scp.click systemctl reload schttp
sleep 0.5
ssh -p2222 root@scp.click systemctl status schttp