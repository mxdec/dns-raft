# dns-raft

DNS cluster using Raft protocol for resource records replication.

The purpose of this case study is to implement the [Raft](https://raft.github.io/) library from [Hashicorp](https://github.com/hashicorp/raft) in order to maintain consistent DNS records across multiple machines.

Any write attempt to KV Store is forwarded to the leader.

```
                             ┌────────┐
                             │zone.txt│
                             └────▲───┘
                                  │
                             read │
                                  │
   ┌──────────┐             ┌─────┴────┐             ┌──────────┐
   │node 02   │             │node 01   │             │node 03   │
   │          │             │          │             │          │
   │follower  ◀─────────────┤leader    ├─────────────▶follower  │
   │          │ replication │          │ replication │          │
   │          │             │          │             │          │
   └──────────┘             └──────────┘             └──────────┘
```

## Build

Compile source code:
```
$ go build -o bin/dns-raft cmd/main.go
```

## Run

Start three nodes:
```
$ bin/dns-raft -id id0 -raft.addr ":8300" -dns.addr ":8600" -zone.file "./zones/z00.txt"
$ bin/dns-raft -id id1 -raft.addr ":8301" -dns.addr ":8601" -zone.file "./zones/z01.txt" -raft.join ":8300"
$ bin/dns-raft -id id2 -raft.addr ":8302" -dns.addr ":8602" -zone.file "./zones/z02.txt" -raft.join ":8300"
```

## DNS

Each node reads its own [zone file](zones/) at execution, and replicates its records to each other.

Resolve addresses from first node:
```
$ dig @127.0.0.1 -p 8600 example.com
$ dig @127.0.0.1 -p 8600 toto.com
$ dig @127.0.0.1 -p 8600 tutu.com
```

Resolve addresses from second node:
```
$ dig @127.0.0.1 -p 8601 example.com
$ dig @127.0.0.1 -p 8601 toto.com
$ dig @127.0.0.1 -p 8601 tutu.com
```

Resolve addresses from third node:
```
$ dig @127.0.0.1 -p 8602 example.com
$ dig @127.0.0.1 -p 8602 toto.com
$ dig @127.0.0.1 -p 8602 tutu.com
```

Add a DNS record to a zone file:
```
echo 'database                 60 A     1.2.3.7' >> zones/z00.txt
```

Reload zone file by sending SIGHUP to node:
```
$ pkill -SIGHUP dns-raft
```

Resolve new address from follower node:
```
$ dig @127.0.0.1 -p 8602 database.example.com
```

## Play with KV Store

Ping the first node:
```
$ echo "kv ping" | nc localhost 8300
PONG
```

Add a key to one of the nodes:
```
$ echo "kv set toto titi" | nc localhost 8300
SUCCESS
$ echo "kv set tata tutu" | nc localhost 8302
SUCCESS
```

Get the value from any node:
```
# first node
$ echo "kv get toto" | nc localhost 8300
titi
$ echo "kv get tata" | nc localhost 8300
tutu

# second node
$ echo "kv get toto" | nc localhost 8301
titi
$ echo "kv get tata" | nc localhost 8301
titi

# third node
$ echo "kv get toto" | nc localhost 8302
titi
$ echo "kv get tata" | nc localhost 8302
titi
```

Remove the key:
```
$ echo "kv del toto" | nc localhost 8300
```

## Inspirations

* http://www.scs.stanford.edu/17au-cs244b/labs/projects/orbay_fisher.pdf
* https://github.com/hashicorp/consul
* https://github.com/otoolep/hraftd
* https://github.com/yongman/leto
