package craft

import (
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var (
	BroadcastMAC = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	ZeroMAC      = net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	ZeroIP       = net.IP{0x00, 0x00, 0x00, 0x00}
)

// SendARPRequest broadcasts an ARP Request with the payload bytes smuggled
// into the Sender Protocol Address (SPA) field.
func SendARPRequest(handle *pcap.Handle, srcMAC net.HardwareAddr, payload []byte) error {
	return sendARP(handle, srcMAC, payload, layers.ARPRequest)
}

// SendARPReply broadcasts an ARP Reply with the payload bytes smuggled
// into the Sender Protocol Address (SPA) field.
func SendARPReply(handle *pcap.Handle, srcMAC net.HardwareAddr, payload []byte) error {
	return sendARP(handle, srcMAC, payload, layers.ARPReply)
}

func sendARP(handle *pcap.Handle, srcMAC net.HardwareAddr, payload []byte, operation uint16) error {
	spa := make([]byte, 4)
	copy(spa, payload)

	eth := &layers.Ethernet{
		SrcMAC:       srcMAC,
		DstMAC:       BroadcastMAC,
		EthernetType: layers.EthernetTypeARP,
	}

	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         operation,
		SourceHwAddress:   []byte(srcMAC),
		SourceProtAddress: spa,
		DstHwAddress:      []byte(ZeroMAC),
		TargetProtAddress: []byte(ZeroIP),
	}

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buf, opts, eth, arp); err != nil {
		return fmt.Errorf("serialize: %w", err)
	}

	return handle.WritePacketData(buf.Bytes())
}

// OpenHandle opens a pcap live capture handle on the given network device.
// promiscuous enables promiscuous mode for sniffing all traffic on the segment.
func OpenHandle(device string, promiscuous bool) (*pcap.Handle, error) {
	return pcap.OpenLive(device, 1600, promiscuous, pcap.BlockForever)
}
