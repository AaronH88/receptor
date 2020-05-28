package netceptor

import (
	"fmt"
	"github.org/ghjm/sockceptor/pkg/debug"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

// UdpDialer implements Backend for outbound UDP
type UdpDialer struct {
	address *net.UDPAddr
}

func NewUdpDialer(address string) (*UdpDialer, error) {
	ua, err := net.ResolveUDPAddr("udp", address); if err != nil {
		return nil, err
	}
	nd := UdpDialer{
		address: ua,
	}
	return &nd, nil
}

func (b *UdpDialer) Start(bsf BackendSessFunc, errf ErrorFunc) {
	go func() {
		for {
			conn, err := net.DialUDP("udp", nil, b.address)
			if err == nil {
				ns := UdpDialerSession{
					conn: conn,
				}
				err = bsf(&ns)
			}
			operr, ok := err.(*net.OpError)
			if ok {
				syserr, ok := operr.Err.(*os.SyscallError)
				if ok {
					if syserr.Err == syscall.ECONNREFUSED {
						// If the other end isn't listening, just keep trying
						time.Sleep(5 * time.Second)
						continue
					}
				}
			}
			errf(err)
			return
		}
	}()
}

//UdpDialerSession implements BackendSession for UDPDialer
type UdpDialerSession struct {
	conn *net.UDPConn
}

func (ns *UdpDialerSession) Send(data []byte) error {
	n, err := ns.conn.Write(data)
	debug.Tracef("UDP sent data %s len %d sent %d err %s\n", data, len(data), n, err)
	if err != nil {
		return err
	}
	if n != len(data) {
		return fmt.Errorf("partial data sent")
	}
	return nil
}

func (ns *UdpDialerSession) Recv() ([]byte, error) {
	buf := make([]byte, MTU)
	n, err := ns.conn.Read(buf)
	debug.Tracef("UDP sending data %s len %d sent %d err %s\n", buf, len(buf), n, err)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (ns *UdpDialerSession) Close() error {
	return ns.conn.Close()
}

// UdpListener implements Backend for inbound UDP
type UdpListener struct {
	laddr           *net.UDPAddr
	conn            *net.UDPConn
	sessChan        chan *UdpListenerSession
	sessRegLock		sync.Mutex
	sessionRegistry map[string]*UdpListenerSession
}

func NewUdpListener(address string) (*UdpListener, error) {
	addr, err := net.ResolveUDPAddr("udp", address); if err != nil {
		return nil, err
	}
	uc, err := net.ListenUDP("udp", addr); if err != nil {
		return nil, err
	}
	ul := UdpListener{
		laddr:           addr,
		conn:            uc,
		sessChan:        make(chan *UdpListenerSession),
		sessRegLock:	 sync.Mutex{},
		sessionRegistry: make(map[string]*UdpListenerSession),
	}
	return &ul, nil
}

func (b *UdpListener) Start(bsf BackendSessFunc, errf ErrorFunc) {
	go func() {
		buf := make([]byte, MTU)
		for {
			n, addr, err := b.conn.ReadFromUDP(buf)
			data := make([]byte, n)
			copy(data, buf)
			debug.Tracef("UDP received data %s len %d err %s\n", data, n, err)
			if err != nil {
				errf(err)
				return
			}
			addrStr := addr.String()
			b.sessRegLock.Lock()
			sess, ok := b.sessionRegistry[addrStr]
			if !ok {
				debug.Printf("Creating new UDP listener session for %s\n", addrStr)
				sess = &UdpListenerSession{
					li:       b,
					raddr:    addr,
					recvChan: make(chan []byte),
				}
				b.sessionRegistry[addrStr] = sess
				b.sessRegLock.Unlock()
				go func () {
					err := bsf(sess); if err != nil {
						errf(err)
					}
				}()
			} else {
				b.sessRegLock.Unlock()
			}
			sess.recvChan <- data
		}
	}()
}

//UdpListenerSession implements BackendSession for UDPListener
type UdpListenerSession struct {
	li *UdpListener
	raddr *net.UDPAddr
	recvChan chan []byte
}

func (ns *UdpListenerSession) Send(data []byte) error{
	n, err := ns.li.conn.WriteToUDP(data, ns.raddr)
	debug.Tracef("UDP sent data %s len %d sent %d err %s\n", data, len(data), n, err)
	if err != nil {
		return err
	} else if n != len(data) {
		return fmt.Errorf("partial data sent")
	} else {
		return nil
	}
}

func (ns *UdpListenerSession) Recv() ([]byte, error) {
	data := <- ns.recvChan
	return data, nil
}

func (ns *UdpListenerSession) Close() error {
	ns.li.sessRegLock.Lock()
	defer ns.li.sessRegLock.Unlock()
	delete(ns.li.sessionRegistry, ns.raddr.String())
	return nil
}
