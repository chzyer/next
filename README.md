# next
[![build](https://travis-ci.org/chzyer/next.svg)](https://travis-ci.org/chzyer/next)

### Server

```shell
$ next sysenv #
$ next genkey
617e819c1551a6be8e31b76ed5cb8157
$ next server -httpkey 617e819c1551a6be8e31b76ed5cb8157
... running...
```

add a user

```shell
$ next shell
Next Server CLI
 -> user add <userName>
password:
```

### Client

```shell
$ next client -aeskey 617e819c1551a6be8e31b76ed5cb8157 -username <userName> -password <password> <serverHost>
```

### test

```shell
ping 10.8.0.1
PING 10.8.0.1 (10.8.0.1) 56(84) bytes of data.
64 bytes from 10.8.0.1: icmp_seq=1 ttl=64 time=1.07 ms
64 bytes from 10.8.0.1: icmp_seq=2 ttl=64 time=0.971 ms
64 bytes from 10.8.0.1: icmp_seq=3 ttl=64 time=1.41 ms
64 bytes from 10.8.0.1: icmp_seq=4 ttl=64 time=1.47 ms
```

### route table
```
$ next shell
Next Client CLI
 -> route add 8.8.8.8/32 'google dns'
route item '8.8.8.8/32' added
 -> route show
Item:
	8.8.8.8/32	google dns
 -> ^D

$ netstat -nr | grep 8.8.8.8
Destination     Gateway         Genmask         Flags   MSS Window  irtt Iface
8.8.8.8         0.0.0.0         255.255.255.255 UH        0 0          0 utun0
```

