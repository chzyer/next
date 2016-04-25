# next
[![build](https://travis-ci.org/chzyer/next.svg)](https://travis-ci.org/chzyer/next)

### install

```shell
$ go get github.com/chzyer/next
```

### server

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

### client

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
```shell
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

### show speed
```shell
$ watch -n 1 next shell dchan speed

Every 1.0s: next shell dchan speed

upload:   66B/s
download: 84B/s
```

### show data channels
```shell
$ watch -n 1 next shell dchan useful
[1.1.1.1:63742 -> 2.2.2.2:42066]: avg: 34ms 52ms 56ms, drop: 0/894 0/294 0/54
[1.1.1.1:63741 -> 2.2.2.2:50701]: avg: 34ms 52ms 56ms, drop: 0/894 0/294 0/54
[1.1.1.1:63738 -> 2.2.2.2:42320]: avg: 36ms 55ms 56ms, drop: 0/894 0/294 0/54
[1.1.1.1:63737 -> 2.2.2.2:47205]: avg: 36ms 54ms 55ms, drop: 0/894 0/294 0/54
[1.1.1.1:63734 -> 2.2.2.2:47205]: avg: 36ms 56ms 55ms, drop: 0/894 0/294 0/54

# avg: mean value of roundtrip time, (15min, 5min, 1min)
# drop: packet count which is droped by remote. (droped packet / total packet)
```

