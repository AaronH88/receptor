package services

import (
	"fmt"
	"net"

	"github.com/ansible/receptor/pkg/logger"
	"github.com/ansible/receptor/pkg/netceptor"
	"github.com/ansible/receptor/pkg/utils"
	"github.com/ghjm/cmdline"
	"github.com/spf13/viper"
)

// UDPProxyServiceInbound listens on a UDP port and forwards packets to a remote Receptor service.
func UDPProxyServiceInbound(s *netceptor.Netceptor, host string, port int, node string, service string) error {
	connMap := make(map[string]netceptor.PacketConner)
	buffer := make([]byte, utils.NormalBufferSize)

	addrStr := fmt.Sprintf("%s:%d", host, port)
	udpAddr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		return fmt.Errorf("could not resolve address %s", addrStr)
	}

	uc, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("error listening on UDP: %s", err)
	}

	ncAddr := s.NewAddr(node, service)

	go func() {
		for {
			n, addr, err := uc.ReadFrom(buffer)
			if err != nil {
				s.Logger.Error("Error reading from UDP: %s\n", err)

				return
			}
			raddrStr := addr.String()
			pc, ok := connMap[raddrStr]
			if !ok {
				pc, err = s.ListenPacket("")
				if err != nil {
					s.Logger.Error("Error listening on Receptor network: %s\n", err)

					return
				}
				s.Logger.Debug("Received new UDP connection from %s\n", raddrStr)
				connMap[raddrStr] = pc
				go runNetceptorToUDPInbound(pc, uc, addr, s.NewAddr(node, service), s.Logger)
			}
			wn, err := pc.WriteTo(buffer[:n], ncAddr)
			if err != nil {
				s.Logger.Error("Error sending packet on Receptor network: %s\n", err)

				continue
			}
			if wn != n {
				s.Logger.Debug("Not all bytes written on Receptor network\n")

				continue
			}
		}
	}()

	return nil
}

func runNetceptorToUDPInbound(pc netceptor.PacketConner, uc *net.UDPConn, udpAddr net.Addr, expectedAddr netceptor.Addr, logger *logger.ReceptorLogger) {
	buf := make([]byte, utils.NormalBufferSize)
	for {
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			logger.Error("Error reading from Receptor network: %s\n", err)

			continue
		}
		if addr != expectedAddr {
			logger.Debug("Received packet from unexpected source %s\n", addr)

			continue
		}
		wn, err := uc.WriteTo(buf[:n], udpAddr)
		if err != nil {
			logger.Error("Error sending packet via UDP: %s\n", err)

			continue
		}
		if wn != n {
			logger.Debug("Not all bytes written via UDP\n")

			continue
		}
	}
}

// UDPProxyServiceOutbound listens on the Receptor network and forwards packets via UDP.
func UDPProxyServiceOutbound(s *netceptor.Netceptor, service string, address string) error {
	connMap := make(map[string]*net.UDPConn)
	buffer := make([]byte, utils.NormalBufferSize)
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return fmt.Errorf("could not resolve UDP address %s", address)
	}
	pc, err := s.ListenPacketAndAdvertise(service, map[string]string{
		"type":    "UDP Proxy",
		"address": address,
	})
	if err != nil {
		return fmt.Errorf("error listening on service %s: %s", service, err)
	}
	go func() {
		for {
			n, addr, err := pc.ReadFrom(buffer)
			if err != nil {
				s.Logger.Error("Error reading from Receptor network: %s\n", err)

				return
			}
			raddrStr := addr.String()
			uc, ok := connMap[raddrStr]
			if !ok {
				uc, err = net.DialUDP("udp", nil, udpAddr)
				if err != nil {
					s.Logger.Error("Error connecting via UDP: %s\n", err)

					return
				}
				s.Logger.Debug("Opened new UDP connection to %s\n", raddrStr)
				connMap[raddrStr] = uc
				go runUDPToNetceptorOutbound(uc, pc, addr, s.Logger)
			}
			wn, err := uc.Write(buffer[:n])
			if err != nil {
				s.Logger.Error("Error writing to UDP: %s\n", err)

				continue
			}
			if wn != n {
				s.Logger.Debug("Not all bytes written to UDP\n")

				continue
			}
		}
	}()

	return nil
}

func runUDPToNetceptorOutbound(uc *net.UDPConn, pc netceptor.PacketConner, addr net.Addr, logger *logger.ReceptorLogger) {
	buf := make([]byte, utils.NormalBufferSize)
	for {
		n, err := uc.Read(buf)
		if err != nil {
			logger.Error("Error reading from UDP: %s\n", err)

			return
		}
		wn, err := pc.WriteTo(buf[:n], addr)
		if err != nil {
			logger.Error("Error writing to the Receptor network: %s\n", err)

			continue
		}
		if wn != n {
			logger.Debug("Not all bytes written to the Netceptor network\n")

			continue
		}
	}
}

// udpProxyInboundCfg is the cmdline configuration object for a UDP inbound proxy.
type UDPProxyInboundCfg struct {
	Port          int    `required:"true" description:"Local UDP port to bind to"`
	BindAddr      string `description:"Address to bind UDP listener to" default:"0.0.0.0"`
	RemoteNode    string `required:"true" description:"Receptor node to connect to"`
	RemoteService string `required:"true" description:"Receptor service name to connect to"`
}

// Run runs the action.
func (cfg UDPProxyInboundCfg) Run() error {
	netceptor.MainInstance.Logger.Debug("Running UDP inbound proxy service %v\n", cfg)

	return UDPProxyServiceInbound(netceptor.MainInstance, cfg.BindAddr, cfg.Port, cfg.RemoteNode, cfg.RemoteService)
}

// udpProxyOutboundCfg is the cmdline configuration object for a UDP outbound proxy.
type UDPProxyOutboundCfg struct {
	Service string `required:"true" description:"Receptor service name to bind to"`
	Address string `required:"true" description:"Address for outbound UDP connection"`
}

// Run runs the action.
func (cfg UDPProxyOutboundCfg) Run() error {
	netceptor.MainInstance.Logger.Debug("Running UDP outbound proxy service %s\n", cfg)

	return UDPProxyServiceOutbound(netceptor.MainInstance, cfg.Service, cfg.Address)
}

func init() {
	version := viper.GetInt("version")
	if version > 1 {
		return
	}
	cmdline.RegisterConfigTypeForApp("receptor-proxies",
		"udp-server", "Listen for UDP and forward via Receptor", UDPProxyInboundCfg{}, cmdline.Section(servicesSection))
	cmdline.RegisterConfigTypeForApp("receptor-proxies",
		"udp-client", "Listen on a Receptor service and forward via UDP", UDPProxyOutboundCfg{}, cmdline.Section(servicesSection))
}
