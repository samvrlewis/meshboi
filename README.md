# ![logo](https://user-images.githubusercontent.com/3880246/124463187-dd916d00-ddd5-11eb-8e0a-923629365637.png) meshboi 

meshboi is a toy mesh VPN implementation, created for fun and learning purposes. It allows the creation of peer to peer networks over the internet in a similar fashion to tools such as [Nebula](https://github.com/slackhq/nebula) and [Tailscale](https://tailscale.com/).

More information about how meshboi works is available on my blog post [Creating a mesh VPN tool for fun and learning](https://www.samlewis.me/2021/07/creating-mesh-vpn-tool-for-fun/).

## Quick Start

1. Download the most recent [release](https://github.com/samvrlewis/meshboi/releases).
2. Start meshboi on one host:

```
./meshboi client -rolodex-address rolodex.samlewis.me -vpn-ip 192.168.50.1/24 -psk <a secure password> -network <a unique network name>
```

3. Start meshboi on another host:

```
./meshboi client -rolodex-address rolodex.samlewis.me -vpn-ip 192.168.50.2/24 -psk <same password as step 2> -network <same network name as part 2>
```

4. The hosts should now be able to communicate as though they were on the same LAN!

Note that this will use the publicly accessible rolodex server that I host. No user data flows through this server other than metadata that contains the internet IP and ports of your instances (though this has not been properly audited, so please use at your own risk!). You are also free to host your own Rolodex server on an an internet accessible server (a cheap EC2 instance or equivalent will work fine). You can do so with:

```
./meshboi rolodex
```

And then use the IP address or hostname of this server when starting meshboi in client mode (with the `-rolodex-address` option).

## Demo

An asciinema recording of meshboi in action:

[![asciinema](https://user-images.githubusercontent.com/3880246/124463198-e124f400-ddd5-11eb-94e9-23de8797137f.png)](https://asciinema.org/a/Cux2gxc8VusS0QbL3tkmWLFb4)