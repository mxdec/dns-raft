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
               -tcp.addr ":5370" \
               -dns.addr ":5450" \
               -raft.addr ":15370" \
               -zone.file "./zones/zone.txt"
```

Start second node:
```
$ bin/dns-raft -id id2 \
               -tcp.addr ":5371" \
               -dns.addr ":5451" \
               -raft.addr ":15371" \
               -raft.join "127.0.0.1:5370"
```

Start third node:
```
$ bin/dns-raft -id id3 \
               -tcp.addr ":5372" \
               -dns.addr ":5452" \
               -raft.addr ":15372" \
               -raft.join "127.0.0.1:5370"
```

## DNS

Resources records are loaded from [zone file](zones/zone.txt) at execution.

Resolve address from first node:
```
$ dig @127.0.0.1 -p 5450 example.com
```

Resolve address from second node:
```
$ dig @127.0.0.1 -p 5451 example.com
```

Resolve address from third node:
```
$ dig @127.0.0.1 -p 5452 example.com
```

## Play with KV Store

Ping the first node:
```
$ echo "ping" | nc localhost 5370
PONG
```

Add a key:
```
$ echo "set toto titi" | nc localhost 5370
SUCCESS
```

Get a key from any node:
```
$ echo "get toto" | nc localhost 5370
titi
$ echo "get toto" | nc localhost 5371
titi
$ echo "get toto" | nc localhost 5372
titi
```

Remove the key:
```
$ echo "del toto" | nc localhost 5370
```

## Inspirations

* http://www.scs.stanford.edu/17au-cs244b/labs/projects/orbay_fisher.pdf
* https://github.com/otoolep/hraftd
* https://github.com/yongman/leto