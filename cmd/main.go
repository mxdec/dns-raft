package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/maxdrc/dns-raft/dns"
	"github.com/maxdrc/dns-raft/store"
	"github.com/maxdrc/dns-raft/tcp"
)

var (
	tcpaddr  string
	dnsaddr  string
	raftaddr string
	raftjoin string
	raftid   string
	zonefile string
)

func init() {
	flag.StringVar(&tcpaddr, "tcp.addr", ":8080", "TCP listen address")
	flag.StringVar(&dnsaddr, "dns.addr", ":5350", "DNS listen address")
	flag.StringVar(&raftaddr, "raft.addr", ":15370", "Raft bus transport bind address")
	flag.StringVar(&raftjoin, "raft.join", "", "Join to already exist cluster")
	flag.StringVar(&raftid, "id", "", "node id")
	flag.StringVar(&zonefile, "zone.file", "", "Zone file containing resource records")
}

func main() {
	flag.Parse()

	quitCh := make(chan int)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	kvs := store.InitStore(raftaddr, raftjoin, raftid)
	tcp := tcp.NewTCP(kvs, tcpaddr)
	tcp.Start()
	dns := dns.NewDNS(kvs, dnsaddr)
	dns.Start()
	dns.InitZone(zonefile)
	go handleSignals(dns, sigCh, quitCh)
	code := <-quitCh
	os.Exit(code)
}

func handleSignals(dns *dns.DNS, sigCh chan os.Signal, quitCh chan int) {
	for {
		s := <-sigCh
		switch s {
		case syscall.SIGHUP:
			dns.LoadZone(zonefile)
		default:
			quitCh <- 0
		}
	}
}
