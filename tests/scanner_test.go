package tests

//! requires ADMIN privledges to run
import (
	"log"
	"net"
	"testing"

	"github.com/google/gopacket/examples/util"
	"github.com/google/gopacket/routing"
	"github.com/jasonkolodziej/go-castv2/scanner"
)

func Test_Scanner(t *testing.T) {
	defer util.Run()()
	router, err := routing.New()
	if err != nil {
		log.Fatal("routing error:", err)
	}
	// for _, arg := range flag.Args() {
	// 	var ip net.IP
	ip := net.ParseIP("192.168.2.152")
	if ip == nil {
		t.Fatal("HELP")
	} else if ip = ip.To4(); ip == nil {
		t.Logf("non-ipv4 target: %q", ip)
		// continue
	}
	// Note:  newScanner creates and closes a pcap Handle once for
	// every scan target.  We could do much better, were this not an
	// example ;)
	s, err := scanner.NewScanner(ip, router)
	if err != nil {
		t.Logf("unable to create scanner for %v: %v", ip, err)
		// continue
	}
	// if err := s.Scan(); err != nil {
	// 	log.Printf("unable to scan %v: %v", ip, err)
	// }
	mac, err := s.GetHwAddr(scanner.DefaultHwAddrParam)
	if err != nil {
		t.Fatalf("unable to get mac for scanner %v: %v", ip, err)
		// continue
	}
	t.Logf("MAC Addr: %v, for IP: %v", mac, ip)
	s.Close()
}

// }
