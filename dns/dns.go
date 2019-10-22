package dns

import (
	"fmt"
	"log"
	"os"

	"github.com/maxdrc/dns-raft/store"
	"github.com/miekg/dns"
)

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
	if err := d.srv.ListenAndServe(); err != nil {
		d.logger.Println(err.Error())
	}
}

// Shutdown closes DNS server
func (d *DNS) Shutdown() {
	d.logger.Println("closing UDP listener")
	if err := d.srv.Shutdown(); err != nil {
		d.logger.Println(err.Error())
	}
}

// LoadZone load zone file into KV Store when node is elected Leader
func (d *DNS) LoadZone(zoneFile string) {
	if len(zoneFile) > 0 {
		if len(d.kvs.WaitLeader()) > 0 {
			d.parseZone(zoneFile)
		} else {
			d.logger.Println("zonefile: error, no leader")
		}
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
