#!/bin/sh -v

if [[ -z "${4:-}" ]]; then
  echo "Use: addroute.sh othernodeip mask externalip devname" 1>&2
  exit 1
fi

nodeip=${1}
shift
mask=${1}
shift
routeip=${1}
shift
extif=${1}

#ip -6 route add $nodeip/$mask via $routeip dev $extif
ip -6 route add $nodeip/$mask dev $extif