#!/bin/bash -e

DEST="$1"

if [ "$DEST" == "" ]; then
  DEST=`dig files.nagae-memooff.top +short`
fi


TO_IP="192.168.18.251"
PORT="6785"

iptables -t nat -I PREROUTING -p tcp --dport ${PORT} -d ${DEST} -j DNAT --to-destination ${TO_IP}:${PORT}
iptables -t nat -I OUTPUT -p tcp --dport ${PORT} -d ${DEST} -j DNAT --to-destination ${TO_IP}:${PORT}
