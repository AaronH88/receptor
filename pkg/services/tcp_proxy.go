package services

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"

	"github.com/ansible/receptor/pkg/logger"
	"github.com/ansible/receptor/pkg/netceptor"
	"github.com/ansible/receptor/pkg/utils"
	"github.com/ghjm/cmdline"
	"github.com/spf13/viper"
)

type NetcForTCPProxy interface {
	GetLogger() *logger.ReceptorLogger
	Dial(node string, service string, tlscfg *tls.Config) (*netceptor.Conn, error)
	ListenAndAdvertise(service string, tlscfg *tls.Config, tags map[string]string) (*netceptor.Listener, error)
	Status() netceptor.Status
	GetClientTLSConfig(name string, expectedHostName string, expectedHostNameType netceptor.ExpectedHostnameType) (*tls.Config, error)
}

// Interface for the net library to generate stubs with mockgen.
type NetLib interface {
	Listen(network string, address string) (net.Listener, error)
	Dial(network string, address string) (net.Conn, error)
}

type NetTCPWrapper struct{}

func (n *NetTCPWrapper) Listen(network string, address string) (net.Listener, error) {
	return net.Listen(network, address) //nolint:noctx // interface method does not receive a context
}

func (n *NetTCPWrapper) Dial(network string, address string) (net.Conn, error) {
	return net.Dial(network, address) //nolint:noctx // interface method does not receive a context
}

// Interface for the tls library to generate stubs with mockgen.
type TLSLib interface {
	NewListener(inner net.Listener, config *tls.Config) net.Listener
	Dial(network string, addr string, config *tls.Config) (*tls.Conn, error)
}

type TLSTCPWrapper struct{}

func (n *TLSTCPWrapper) NewListener(inner net.Listener, config *tls.Config) net.Listener {
	return tls.NewListener(inner, config)
}

func (n *TLSTCPWrapper) Dial(network string, addr string, config *tls.Config) (*tls.Conn, error) {
	return tls.Dial(network, addr, config) //nolint:noctx // interface method does not receive a context
}

// Interface for the Net Listener to generate stubs with mockgen.
type NetListenerTCP interface {
	net.Listener
}

// Interface for the utils package to generate stubs with mockgen.
type UtilsLib interface {
	BridgeConns(c1 io.ReadWriteCloser, c1Name string, c2 io.ReadWriteCloser, c2Name string, logger *logger.ReceptorLogger)
}

type UtilsTCPWrapper struct{}

func (u *UtilsTCPWrapper) BridgeConns(c1 io.ReadWriteCloser, c1Name string, c2 io.ReadWriteCloser, c2Name string, logger *logger.ReceptorLogger) {
	utils.BridgeConns(c1, c1Name, c2, c2Name, logger)
}

// Interface to mock the Connection object returned from Accept.
type TCPConn interface {
	net.Conn
}

type tcpInboundRoute struct {
	staticNode    string
	staticService string
	selector      map[string]string
	nodeFilter    string
	serviceFilter string
	tlsClientName string
	picker        RoundRobinPicker
}

func (r tcpInboundRoute) usesSelector() bool {
	return len(r.selector) > 0
}

func (r tcpInboundRoute) pickEndpoint(status netceptor.Status) (ServiceEndpoint, error) {
	if !r.usesSelector() {
		return ServiceEndpoint{NodeID: r.staticNode, Service: r.staticService}, nil
	}

	endpoints := DiscoverServicesByTags(status.Advertisements, r.selector, r.nodeFilter, r.serviceFilter)
	if len(endpoints) == 0 {
		return ServiceEndpoint{}, fmt.Errorf("no mesh services match selector %v", r.selector)
	}

	return r.picker.Next(endpoints), nil
}

// TCPProxyServiceInbound listens on a TCP port and forwards the connection over the Receptor network.
func TCPProxyServiceInbound(s NetcForTCPProxy, host string, port int, tlsServer *tls.Config,
	route tcpInboundRoute, netTCP NetLib, tlsTCP TLSLib, utilsTCP UtilsLib,
) error {
	tli, err := netTCP.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if tlsServer != nil {
		tli = tlsTCP.NewListener(tli, tlsServer)
	}
	if err != nil {
		return fmt.Errorf("error listening on TCP: %s", err)
	}
	go func() {
		for {
			tc, err := tli.Accept()
			if err != nil {
				s.GetLogger().Error("error accepting TCP connection: %s\n", err)

				return
			}
			var endpoint ServiceEndpoint
			if route.usesSelector() {
				var pickErr error
				endpoint, pickErr = route.pickEndpoint(s.Status())
				if pickErr != nil {
					s.GetLogger().Error("error selecting tcp-server backend: %s\n", pickErr)
					_ = tc.Close()

					continue
				}
			} else {
				endpoint = ServiceEndpoint{NodeID: route.staticNode, Service: route.staticService}
			}

			var tlsClientCfg *tls.Config
			if route.tlsClientName != "" {
				tlsClientCfg, err = s.GetClientTLSConfig(
					route.tlsClientName,
					endpoint.NodeID,
					netceptor.ExpectedHostnameTypeReceptor,
				)
				if err != nil {
					s.GetLogger().Error("error loading TLS client config for %s: %s\n", endpoint.NodeID, err)
					_ = tc.Close()

					continue
				}
			}

			qc, err := s.Dial(endpoint.NodeID, endpoint.Service, tlsClientCfg)
			if err != nil {
				s.GetLogger().Error("error connecting on Receptor network to %s/%s: %s\n",
					endpoint.NodeID, endpoint.Service, err)
				_ = tc.Close()

				continue
			}
			go utilsTCP.BridgeConns(tc, "tcp service", qc, "receptor connection", s.GetLogger())
		}
	}()

	return nil
}

func mergeTCPProxyTags(address string, userTags map[string]string) map[string]string {
	tags := map[string]string{
		"type":    "TCP Proxy",
		"address": address,
	}
	for key, value := range userTags {
		tags[key] = value
	}

	return tags
}

// TCPProxyServiceOutbound listens on the Receptor network and forwards the connection via TCP.
func TCPProxyServiceOutbound(s NetcForTCPProxy, service string, tlsServer *tls.Config,
	address string, tlsClient *tls.Config, userTags map[string]string, netTCP NetLib, tlsTCP TLSLib, utilsTCP UtilsLib,
) error {
	qli, err := s.ListenAndAdvertise(service, tlsServer, mergeTCPProxyTags(address, userTags))
	if err != nil {
		return fmt.Errorf("error listening on Receptor network: %s", err)
	}
	go func() {
		for {
			qc, err := qli.Accept()
			if err != nil {
				s.GetLogger().Error("Error accepting connection on Receptor network: %s\n", err)

				return
			}
			var tc net.Conn
			if tlsClient == nil {
				tc, err = netTCP.Dial("tcp", address)
			} else {
				tc, err = tlsTCP.Dial("tcp", address, tlsClient)
			}
			if err != nil {
				s.GetLogger().Error("Error connecting via TCP: %s\n", err)

				continue
			}
			go utilsTCP.BridgeConns(qc, "receptor service", tc, "tcp connection", s.GetLogger())
		}
	}()

	return nil
}

// tcpProxyInboundCfg is the cmdline configuration object for a TCP inbound proxy.
type TCPProxyInboundCfg struct {
	Port          int               `required:"true" description:"Local TCP port to bind to"`
	BindAddr      string            `description:"Address to bind TCP listener to" default:"0.0.0.0"`
	RemoteNode    string            `description:"Receptor node to connect to (static mode) or optional scope filter with selector"`
	RemoteService string            `description:"Receptor service name to connect to (static mode) or optional scope filter with selector"`
	Selector      map[string]string `description:"Label selector for dynamic backend discovery from mesh service advertisements"`
	TLSServer     string            `description:"Name of TLS server config for the TCP listener"`
	TLSClient     string            `description:"Name of TLS client config for the Receptor connection"`
}

func (cfg TCPProxyInboundCfg) route() (tcpInboundRoute, error) {
	hasSelector := len(cfg.Selector) > 0
	hasStatic := cfg.RemoteNode != "" && cfg.RemoteService != ""

	switch {
	case hasSelector:
		return tcpInboundRoute{
			selector:      cfg.Selector,
			nodeFilter:    cfg.RemoteNode,
			serviceFilter: cfg.RemoteService,
			tlsClientName: cfg.TLSClient,
		}, nil
	case hasStatic:
		return tcpInboundRoute{
			staticNode:    cfg.RemoteNode,
			staticService: cfg.RemoteService,
			tlsClientName: cfg.TLSClient,
		}, nil
	default:
		return tcpInboundRoute{}, fmt.Errorf("tcp-server requires selector and/or remotenode+remoteservice")
	}
}

// Run runs the action.
func (cfg TCPProxyInboundCfg) Run() error {
	netceptor.MainInstance.Logger.Debug("Running TCP inbound proxy service %v\n", cfg)
	route, err := cfg.route()
	if err != nil {
		return err
	}
	if cfg.TLSClient != "" && !route.usesSelector() {
		_, err = netceptor.MainInstance.GetClientTLSConfig(
			cfg.TLSClient,
			route.staticNode,
			netceptor.ExpectedHostnameTypeReceptor,
		)
		if err != nil {
			return err
		}
	}
	TLSServerConfig, err := netceptor.MainInstance.GetServerTLSConfig(cfg.TLSServer)
	if err != nil {
		return err
	}

	return TCPProxyServiceInbound(netceptor.MainInstance, cfg.BindAddr, cfg.Port, TLSServerConfig,
		route, &NetTCPWrapper{}, &TLSTCPWrapper{}, &UtilsTCPWrapper{})
}

// tcpProxyOutboundCfg is the cmdline configuration object for a TCP outbound proxy.
type TCPProxyOutboundCfg struct {
	Service   string            `required:"true" description:"Receptor service name to bind to"`
	Address   string            `required:"true" description:"Address for outbound TCP connection"`
	Tags      map[string]string `description:"Optional key/value tags advertised with this service"`
	TLSServer string            `description:"Name of TLS server config for the Receptor service"`
	TLSClient string            `description:"Name of TLS client config for the TCP connection"`
}

// Run runs the action.
func (cfg TCPProxyOutboundCfg) Run() error {
	netceptor.MainInstance.Logger.Debug("Running TCP outbound proxy service %s\n", cfg)
	TLSServerConfig, err := netceptor.MainInstance.GetServerTLSConfig(cfg.TLSServer)
	if err != nil {
		return err
	}
	host, _, err := net.SplitHostPort(cfg.Address)
	if err != nil {
		return err
	}
	tlsClientCfg, err := netceptor.MainInstance.GetClientTLSConfig(cfg.TLSClient, host, netceptor.ExpectedHostnameTypeDNS)
	if err != nil {
		return err
	}

	return TCPProxyServiceOutbound(netceptor.MainInstance, cfg.Service, TLSServerConfig, cfg.Address, tlsClientCfg, cfg.Tags,
		&NetTCPWrapper{}, &TLSTCPWrapper{}, &UtilsTCPWrapper{})
}

func init() {
	version := viper.GetInt("version")
	if version > 1 {
		return
	}
	cmdline.RegisterConfigTypeForApp("receptor-proxies",
		"tcp-server", "Listen for TCP and forward via Receptor", TCPProxyInboundCfg{}, cmdline.Section(servicesSection))
	cmdline.RegisterConfigTypeForApp("receptor-proxies",
		"tcp-client", "Listen on a Receptor service and forward via TCP", TCPProxyOutboundCfg{}, cmdline.Section(servicesSection))
}
