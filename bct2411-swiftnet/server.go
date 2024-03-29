package main

import (
	"log"
	"net"
	"syscall"
	"tun"
)

var other_ends []*net.UDPAddr

func register_begin() {
	other_ends = make([]*net.UDPAddr, 0)
}

func register_connection(ra *net.UDPAddr) {
	log.Print("Got registration from ", ra)
	other_ends = append(other_ends, ra)
}

func server() {
	tun_ip := &net.IP{192, 168, 7, 2}
	log.Print("External IP is ", tun_ip.String())

	log.Print("Initializing tun device")
	tundev, err := tun.Tun_alloc(tun.IFF_TUN | tun.IFF_NO_PI)
	if err != nil {
		log.Print("Could not allocate tun device")
		log.Fatal(err)
	}
	// assume that normal MTU is 1500
	err = tun.SetMTU(tundev.Name(), 1500-TUNNEL_OVERHEAD)
	if err != nil {
		log.Print("Could not set MTU: ", err)
	}
	log.Print("Opened up tun device " + tundev.Name())

	log.Print("Configuring device with ifconfig")
	err = tun.Ifconfig(tundev.Name(), TTT_SERVER_IP, TTT_CLIENT_IP)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Listening on 0.0.0.0:", SERVER_PORT)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: SERVER_PORT})
	if err != nil {
		log.Fatal(err)
	}

	// set up listening channels for udp and tun
	tunchan := make(chan []byte)
	udpchan := make(chan UDPRecv)

	go listenTun(tundev, tunchan)
	go listenUDP(conn, udpchan)

	for {
		select {
		case tun_pkt, ok := <-tunchan:
			if !ok {
				log.Fatal("Error reading from tun")
			}
			log.Printf("Got %d bytes from tundev", len(tun_pkt))
			if other_ends != nil {
				log.Printf("Sending %d bytes to %s", len(tun_pkt), other_ends[int(packet_seq)%len(other_ends)])
				forward_packet(conn, other_ends[int(packet_seq)%len(other_ends)], tun_pkt)
			} else {
				log.Print("Got data without registration")
			}
		// listenUDP sends a struct with byte count and remote_addr
		case udpr, ok := <-udpchan:
			if !ok {
				log.Fatal("Error reading from udp")
			}
			envelope := udpr.Data
			remote_addr := udpr.RemoteAddr

			switch envelope[0] {
			case TTT_DATA: // packet to be forwarded
				pkt := envelope[ENVELOPE_LENGTH:]
				log.Printf("Got %d bytes from %s addressed to %s",
					len(envelope), remote_addr,
					get_ip_dest(pkt))

				log.Printf("Sending through tun device")
				/* this breaks routing
				replace_src_addr(pkt, *tun_ip)
				ReplaceIPHeaderChecksum(pkt)
				*/
				tundev.Write(pkt)
			case TTT_REGISTER_BEGIN: // clear existing registrations
				register_begin()
				_, err = conn.WriteToUDP([]byte{TTT_REGISTER_BEGIN}, remote_addr)
				if err != nil {
					log.Print(err)
				}
				log.Print("Clearing registrations ...")
			case TTT_REGISTER: // registration
				log.Print("Received registration from ", remote_addr)
				register_connection(remote_addr)
			default:
				log.Print("Received packet of type ", envelope[0])
			}
		}
	}

}

// Tries to figure out what IP address is the external address
// by looking up Google's DNS service (8.8.8.8) and seeing which
// interface gets used for it
// DEPRECATED: linux will handle routing anyways
func get_ext_addr() (*net.IP, error) {
	googDns := &syscall.SockaddrInet4{
		Port: 53, // doesn't really matter
		Addr: [4]byte{8, 8, 8, 8},
	}
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, err
	}

	err = syscall.Connect(sock, googDns)
	if err != nil {
		return nil, err
	}

	ourname, err := syscall.Getsockname(sock)
	if err != nil {
		return nil, err
	}

	// get addr in [4]byte form
	ipb := (ourname.(*syscall.SockaddrInet4)).Addr
	//ip := make(net.IP, 4)
	ip := net.IPv4(ipb[0], ipb[1], ipb[2], ipb[3])
	return &ip, nil
}
