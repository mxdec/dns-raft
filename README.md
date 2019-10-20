# dns-raft

DNS cluster using Raft protocol for resource records replication.

The purpose of this case study is to implement the [Raft](https://raft.github.io/) library from [Hashicorp](https://github.com/hashicorp/raft) in order to maintain consistent DNS records across multiple machines.

## Build

Compile source code:
```
$ go build -o bin/dns-raft cmd/main.go
```

## Run

Start first node:
```
$ bin/dns-raft -id id1 \
               -tcp.addr ":8080" \
               -dns.addr ":5350" \
               -raft.addr ":15370" \
               -zone.file "./zones/zone.txt"
```

Start second node:
```
$ bin/dns-raft -id id2 \
               -tcp.addr ":8081" \
               -dns.addr ":5351" \
               -raft.addr ":15371" \
               -raft.join "127.0.0.1:8080"
```

Start third node:
```
$ bin/dns-raft -id id3 \
               -tcp.addr ":8082" \
               -dns.addr ":5352" \
               -raft.addr ":15372" \
               -raft.join "127.0.0.1:8080"
```

## DNS

Resources records are loaded from [zone file](zones/zone.txt) at execution.

Resolve address from first node:
```
$ dig @127.0.0.1 -p 5350 example.com
```

Resolve address from second node:
```
$ dig @127.0.0.1 -p 5351 example.com
```

Resolve address from third node:
```
$ dig @127.0.0.1 -p 5352 example.com
```

Add a DNS record:
```
echo 'database                 60 A     1.2.3.7' >> zones/zone.txt
```

Reload zone file by sending SIGHUP to leader node:
```
$ pkill -SIGHUP dns-raft
```

Resolve new address from follower node:
```
$ dig @127.0.0.1 -p 5352 database.example.com
```

## Play with KV Store

Ping the first node:
```
$ echo "ping" | nc localhost 8080
PONG
```

Add a key:
```
$ echo "set toto titi" | nc localhost 8080
SUCCESS
```

Get a key from any node:
```
$ echo "get toto" | nc localhost 8080
titi
$ echo "get toto" | nc localhost 8081
titi
$ echo "get toto" | nc localhost 8082
titi
```

Remove the key:
```
$ echo "del toto" | nc localhost 8080
```

## Inspirations

* http://www.scs.stanford.edu/17au-cs244b/labs/projects/orbay_fisher.pdf
* https://github.com/otoolep/hraftd
* https://github.com/yongman/leto