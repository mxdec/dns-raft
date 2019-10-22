package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxdrc/dns-raft/dns"
	"github.com/maxdrc/dns-raft/store"
)

var (
	dnsaddr  string
	raftaddr string
	raftjoin string
	raftid   string
	zonefile string
)

func init() {
	flag.StringVar(&dnsaddr, "dns.addr", ":5350", "DNS listen address")
	flag.StringVar(&raftaddr, "raft.addr", ":15370", "Raft bus transport bind address")
	flag.StringVar(&raftjoin, "raft.join", "", "Join to already exist cluster")
	flag.StringVar(&raftid, "id", "", "node id")
	flag.StringVar(&zonefile, "zone.file", "", "Zone file containing resource records")
}

func main() {
	flag.Parse()

	kvs := store.InitStore(raftaddr, raftjoin, raftid)
	dns := dns.NewDNS(kvs, dnsaddr)
	go handleSignals(kvs, dns)
	dns.LoadZone(zonefile)
	dns.Start()
}

func handleSignals(kvs *store.Store, dns *dns.DNS) {
	signalChan := make(chan os.Signal, 1)
	sighupChan := make(chan os.Signal, 1)

	signal.Notify(sighupChan, syscall.SIGHUP)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-sighupChan:
			dns.LoadZone(zonefile)
		case <-signalChan:
			kvs.Leave(kvs.RaftID)
			dns.Shutdown()
			kvs.Shutdown()
		}
	}
}
