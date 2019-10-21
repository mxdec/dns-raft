# dns-raft

DNS cluster using Raft protocol for resource records replication.

The purpose of this case study is to implement the [Raft](https://raft.github.io/) library from [Hashicorp](https://github.com/hashicorp/raft) in order to maintain consistent DNS records across multiple machines.

Any write attempt to KV Store is forwarded to the leader.

## Design

```
  ┌───────────┐             ┌───────────┐             ┌───────────┐
  │z01.txt    │             │z00.txt    │             │z02.txt    │
  │           │             │           │             │           │
  │bar.com    │             │foo.com    │             │baz.com    │
  └─────▲─────┘             └─────▲─────┘             └─────▲─────┘
        │                         │                         │      
    read│                     read│                     read│      
        │                         │                         │      
  ┌─────┴─────┐             ┌─────┴─────┐             ┌─────┴─────┐
  │           │  leader     │           │     leader  │           │
  │           │forwarding   │           │   forwarding│           │
  │ node 01   ├─────────────▶ node 00   ◀─────────────┤ node 02   │
  │           │             │           │             │           │
  │ follower  │             │ leader    │             │ follower  │
  │           │  replication│           │replication  │           │
  │           ◀─────────────┤           ├─────────────▶           │
  └───────────┘             └───────────┘             └───────────┘
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

Each node reads a [zone file](zones/) at execution, and replicates its records to other nodes.

Resolve addresses from first node:
```
$ dig @127.0.0.1 -p 8600 foo.com
$ dig @127.0.0.1 -p 8600 bar.com
$ dig @127.0.0.1 -p 8600 baz.com
```

Resolve addresses from second node:
```
$ dig @127.0.0.1 -p 8601 foo.com
$ dig @127.0.0.1 -p 8601 bar.com
$ dig @127.0.0.1 -p 8601 baz.com
```

Resolve addresses from third node:
```
$ dig @127.0.0.1 -p 8602 foo.com
$ dig @127.0.0.1 -p 8602 bar.com
$ dig @127.0.0.1 -p 8602 baz.com
```

Add a DNS record to a zone file:
```
echo 'database                 60 A     1.2.3.7' >> zones/z00.txt
```

Reload zone file by sending SIGHUP to node:
```
$ pkill -SIGHUP dns-raft
```

Resolve new address from any node:
```
$ dig @127.0.0.1 -p 8602 database.foo.com
```

## Play with KV Store

Ping the first node:
```
$ echo "kv ping" | nc localhost 8300
PONG
```

Add a key to one of the nodes:
```
$ echo "kv set foo bar" | nc localhost 8301
SUCCESS
```

Get the value from any node:
```
$ echo "kv get foo" | nc localhost 8300
bar
$ echo "kv get foo" | nc localhost 8301
bar
$ echo "kv get foo" | nc localhost 8302
bar
```

Remove the key:
```
$ echo "kv del foo" | nc localhost 8302
SUCCESS
$ echo "kv get foo" | nc localhost 8301
ERROR
```

## Inspirations

* http://www.scs.stanford.edu/17au-cs244b/labs/projects/orbay_fisher.pdf
* https://github.com/hashicorp/consul
* https://github.com/otoolep/hraftd
* https://github.com/yongman/leto
