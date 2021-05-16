#!/usr/bin/env bash

# Sets up a blue NS and a green NS with a veth pair to allow direct
# communications. The idea is that this allows for testing of tipod, with the
# green running in the green namespace and the blue running in the blue
# namespace. The green and blue can communicate directly through the veth
# link.
set -x

trap "exit" INT TERM
trap "pkill -P $$; exit 0" EXIT

go build -o meshboi cmd/meshboi/main.go

ip netns add blue
ip netns add green
ip link add veth0 type veth peer name veth1
ip link set veth0 netns blue
ip link set veth1 netns green

ip netns exec blue ip addr add 10.1.1.1/24 dev veth0
ip netns exec blue ip link set dev veth0 up

ip netns exec green ip addr add 10.1.1.2/24 dev veth1
ip netns exec green ip link set dev veth1 up

ip netns exec blue ip link set lo up
ip netns exec green ip link set lo up

ip netns exec blue ./meshboi rollodex -listen-address 10.1.1.1 &
ip netns exec blue ./meshboi client -rollodex-address 10.1.1.1 -vpn-ip 192.168.50.1/24 -psk testpassword -network testnetwork &
ip netns exec green ./meshboi client -rollodex-address 10.1.1.1 -vpn-ip 192.168.50.2/24 -psk testpassword -network testnetwork &

sleep 30

ip netns exec blue ping -c 1 192.168.50.2

if [[ $? -ne 0 ]] ; then
    echo "Not successful blue to green"
    exit 1
fi

ip netns exec green ping -c 1 192.168.50.1

if [[ $? -ne 0 ]] ; then
    echo "Not successful green to blue"
    exit 1
fi

echo "Success!"

exit 0