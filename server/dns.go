package server

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/maxdcr/pagesjaunes/store"
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
	// key := fmt.Sprintf("%s_%d", r.Header().Name, rr.Header().Rrtype)
	key := fmt.Sprintf("%s_%d", r.Question[0].Name, r.Question[0].Qtype)
	// switch r.Question[0].Qtype {
	// case dns.TypeA:
	msg.Authoritative = true
	v, ok := d.kvs.Get(key)
	if ok {
		if rr, err := dns.NewRR(v); err == nil {
			msg.Answer = append(msg.Answer, rr)
		}
	}
	// }
	w.WriteMsg(&msg)
}

// InitDNS initializes DNS server
func InitDNS(kvs *store.Store, dnsAddr, zoneFile string) {
	var srv dns.Server

	srv.Addr = dnsAddr
	srv.Net = "udp"
	srv.Handler = &dnsHandler{
		kvs:    kvs,
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}

	// load zone file into KV Store if node is Raft Leader
	if len(zoneFile) > 0 {
		select {
		case <-kvs.LeaderCh():
			ld := zoneSeeder{kvs: kvs}
			ld.loadZone(zoneFile)
		case <-time.After(5 * time.Second):
			log.Println("zonefile: error, not leader")
		}
	}

	log.Printf("setting UDP listener to %s", srv.Addr)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("Failed to set UDP listener %s", err.Error())
		}
	}()
}

type zoneSeeder struct {
	kvs *store.Store
}

// LoadZone iterates over entries in the zonefile and creates
// Record_X objects for resource record type X. It adds these entries to the
// raft KV Store for propogation to other DNS nameserver nodes.
func (z *zoneSeeder) loadZone(zoneFile string) {
	f, err := os.Open(zoneFile)
	if err != nil {
		log.Printf("zonefile: error, %v", err)
		return
	}
	defer f.Close()
	zp := dns.NewZoneParser(f, ".", zoneFile)
	for rr, ok := zp.Next(); ok; rr, ok = zp.Next() {
		// generate a unique key to store the Resource Record
		key := fmt.Sprintf("%s_%d", rr.Header().Name, rr.Header().Rrtype)
		// store the serialized record
		if err := z.kvs.Set(key, rr.String()); err != nil {
			log.Printf("error storing record: %v", err)
			continue
		}
		log.Printf("stored: %s\n", rr.String())
	}
	if err := zp.Err(); err != nil {
		log.Printf("error reading zone file: %v", err)
	}
}
