package dns

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/maxdcr/dns-raft/store"
	"github.com/miekg/dns"
)

// DNS wrapper
type DNS struct {
	srv    dns.Server
	kvs    *store.Store
	logger *log.Logger
}

// NewDNS initializes Name Server
func NewDNS(kvs *store.Store, dnsAddr string) *DNS {
	return &DNS{
		kvs:    kvs,
		logger: log.New(os.Stderr, "", log.LstdFlags),
		srv: dns.Server{
			Addr: dnsAddr,
			Net:  "udp",
			Handler: &dnsHandler{
				kvs:    kvs,
				logger: log.New(os.Stderr, "", log.LstdFlags),
			},
		},
	}
}

// Start initializes DNS server
func (d *DNS) Start() {
	d.logger.Printf("setting UDP listener to %s", d.srv.Addr)
	go func() {
		d.logger.Fatal(d.srv.ListenAndServe())
	}()
}

// InitZone load zone file into KV Store when node is elected Leader
func (d *DNS) InitZone(zoneFile string) {
	if len(zoneFile) > 0 {
		select {
		case <-d.kvs.LeaderCh():
			d.parseZone(zoneFile)
		case <-time.After(5 * time.Second):
			d.logger.Println("zonefile: error, not leader")
		}
	}
}

// LoadZone reload zone file into KV Store if node is Raft Leader
func (d *DNS) LoadZone(zoneFile string) {
	if len(zoneFile) > 0 && d.kvs.IsLeader() {
		d.parseZone(zoneFile)
	}
}

// parseZone iterates over entries in the zonefile and creates
// Record_X objects for resource record type X. It adds these entries to the
// raft KV Store for propogation to other DNS nameserver nodes.
func (d *DNS) parseZone(zoneFile string) {
	f, err := os.Open(zoneFile)
	if err != nil {
		d.logger.Printf("zonefile: error, %v", err)
		return
	}
	defer f.Close()
	zp := dns.NewZoneParser(f, ".", zoneFile)
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		// generate a unique key to store the Resource Record
		key := fmt.Sprintf("%s_%d", rr.Header().Name, rr.Header().Rrtype)
		// store the serialized record
		if err := d.kvs.Set(key, rr.String()); err != nil {
			d.logger.Printf("error storing record: %v", err)
			continue
		}
	}
	if err := zp.Err(); err != nil {
		d.logger.Printf("error reading zone file: %v", err)
	}
	d.logger.Println("records loaded into KV Store")
}

type dnsHandler struct {
	kvs    *store.Store
	logger *log.Logger
}

// ServeDNS finds record in the KV Store
func (d *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	var msg dns.Msg

	msg.SetReply(r)
	d.logger.Printf("incoming DNS request from %s", w.RemoteAddr().String())
	key := fmt.Sprintf("%s_%d", r.Question[0].Name, r.Question[0].Qtype)
	msg.Authoritative = true
	v, ok := d.kvs.Get(key)
	if ok {
		if rr, err := dns.NewRR(v); err == nil {
			msg.Answer = append(msg.Answer, rr)
		}
	}
	w.WriteMsg(&msg)
}
